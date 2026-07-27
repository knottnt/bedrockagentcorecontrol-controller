# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for Harness API.
"""

import pytest
import time

from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_bedrockagentcorecontrol_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources

HARNESS_RESOURCE_PLURAL = "harnesses"

UPDATE_WAIT_AFTER_SECONDS = 10
# Harness resources are provisioned asynchronously; allow ample time to reach READY.
SYNC_WAIT_PERIODS = 30


@pytest.fixture(scope="module")
def simple_harness():
    harness_name = random_suffix_name("acktest-harness", 40, delimiter="-")
    # AWS API requires the harness name to start with a letter and contain only
    # alphanumeric characters and underscores (no hyphens).
    harness_spec_name = harness_name.replace("-", "_")

    resources = get_bootstrap_resources()

    replacements = REPLACEMENT_VALUES.copy()
    replacements["HARNESS_NAME"] = harness_name
    replacements["HARNESS_SPEC_NAME"] = harness_spec_name
    replacements["ROLE_ARN"] = resources.MemoryRole.arn
    # Used by the union-field update test to construct a valid CUSTOM_JWT
    # authorizer configuration that points at a real Cognito OIDC discovery
    # endpoint provisioned during bootstrap.
    replacements["DISCOVERY_URL"] = (
        f"https://cognito-idp.{resources.GatewayUserPool.region}.amazonaws.com"
        f"/{resources.GatewayUserPool.user_pool_id}/.well-known/openid-configuration"
    )
    replacements["CLIENT_ID"] = "test-client-id"

    resource_data = load_bedrockagentcorecontrol_resource(
        "harness",
        additional_replacements=replacements,
    )

    ref = k8s.CustomResourceReference(
        CRD_GROUP,
        CRD_VERSION,
        HARNESS_RESOURCE_PLURAL,
        harness_name,
        namespace="default",
    )

    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    yield (ref, cr)

    try:
        _, deleted = k8s.delete_custom_resource(ref, wait_periods=5, period_length=15)
        assert deleted
    except:
        pass


@service_marker
@pytest.mark.canary
class TestHarness:
    def test_create_delete(self, simple_harness, bedrockagentcorecontrol_client):
        (ref, cr) = simple_harness

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        cr = k8s.get_resource(ref)
        harness_id = cr["status"]["id"]
        assert harness_id is not None
        assert cr["status"]["ackResourceMetadata"]["arn"] is not None
        assert cr["status"]["status"] is not None
        assert cr["status"]["harnessVersion"] is not None

        # Verify the harness exists in AWS
        aws_harness = bedrockagentcorecontrol_client.get_harness(
            harnessId=harness_id
        )["harness"]
        assert aws_harness["harnessId"] == harness_id
        assert aws_harness["harnessName"] == cr["spec"]["name"]

    def test_update_max_iterations(self, simple_harness, bedrockagentcorecontrol_client):
        (ref, cr) = simple_harness

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        cr = k8s.get_resource(ref)
        harness_id = cr["status"]["id"]

        # Update a simple mutable scalar field
        updates = {
            "spec": {
                "maxIterations": 20,
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        # Verify in AWS
        aws_harness = bedrockagentcorecontrol_client.get_harness(
            harnessId=harness_id
        )["harness"]
        assert aws_harness["maxIterations"] == 20

    def test_update_authorizer_configuration(self, simple_harness, bedrockagentcorecontrol_client):
        (ref, cr) = simple_harness

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        cr = k8s.get_resource(ref)
        harness_id = cr["status"]["id"]

        resources = get_bootstrap_resources()
        discovery_url = (
            f"https://cognito-idp.{resources.GatewayUserPool.region}.amazonaws.com"
            f"/{resources.GatewayUserPool.user_pool_id}/.well-known/openid-configuration"
        )
        client_id = "test-client-id"

        # Mutate a PATCH-wrapper union field. This exercises the
        # harnessAuthorizerConfigurationToSDK converter and the
        # UpdatedAuthorizerConfiguration wrapper populated in the
        # sdk_update_post_build_request hook. The harness is created without an
        # authorizer, so setting one produces a delta on Spec.AuthorizerConfiguration.
        updates = {
            "spec": {
                "authorizerConfiguration": {
                    "customJWTAuthorizer": {
                        "discoveryURL": discovery_url,
                        "allowedAudience": [client_id],
                        "allowedClients": [client_id],
                    },
                },
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        # Verify the union field reached AWS. get_harness returns the authorizer
        # union as a tagged dict keyed by the member name (customJWTAuthorizer).
        aws_harness = bedrockagentcorecontrol_client.get_harness(
            harnessId=harness_id
        )["harness"]
        aws_authorizer = aws_harness["authorizerConfiguration"]
        assert "customJWTAuthorizer" in aws_authorizer
        aws_jwt = aws_authorizer["customJWTAuthorizer"]
        assert aws_jwt["discoveryUrl"] == discovery_url
        assert client_id in aws_jwt["allowedClients"]
        assert client_id in aws_jwt["allowedAudience"]

    def test_update_tags(self, simple_harness, bedrockagentcorecontrol_client):
        (ref, cr) = simple_harness

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        cr = k8s.get_resource(ref)
        harness_arn = cr["status"]["ackResourceMetadata"]["arn"]

        # Add tags
        updates = {
            "spec": {
                "tags": {
                    "Environment": "test",
                    "Project": "ack-e2e",
                },
            },
        }
        k8s.patch_custom_resource(ref, updates)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        # Verify tags in AWS
        aws_tags = bedrockagentcorecontrol_client.list_tags_for_resource(
            resourceArn=harness_arn
        )["tags"]
        assert aws_tags["Environment"] == "test"
        assert aws_tags["Project"] == "ack-e2e"

        # Remove a tag
        cr = k8s.get_resource(ref)
        cr["spec"]["tags"] = {"Environment": "staging"}
        k8s.replace_custom_resource(ref, cr)
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        assert k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=SYNC_WAIT_PERIODS)

        # Verify tag removal in AWS
        aws_tags = bedrockagentcorecontrol_client.list_tags_for_resource(
            resourceArn=harness_arn
        )["tags"]
        assert aws_tags["Environment"] == "staging"
        assert "Project" not in aws_tags
