package h_arn_ess

import (
	"context"

	svcapitypes "github.com/aws-controllers-k8s/bedrockagentcorecontrol-controller/apis/v1alpha1"
	"github.com/aws-controllers-k8s/bedrockagentcorecontrol-controller/pkg/tags"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"
)

func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) (map[string]*string, error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getTags")
	defer func() { exit(nil) }()

	resp, err := rm.sdkapi.ListTagsForResource(ctx, &svcsdk.ListTagsForResourceInput{
		ResourceArn: &resourceARN,
	})
	rm.metrics.RecordAPICall("GET", "ListTagsForResource", err)
	if err != nil {
		return nil, err
	}
	return aws.StringMap(resp.Tags), nil
}

func (rm *resourceManager) syncTags(
	ctx context.Context,
	desired *resource,
	latest *resource,
) error {
	resourceARN := string(*latest.ko.Status.ACKResourceMetadata.ARN)
	desiredTags := aws.ToStringMap(desired.ko.Spec.Tags)
	existingTags := aws.ToStringMap(latest.ko.Spec.Tags)
	return tags.SyncTags(
		ctx, rm.sdkapi, rm.metrics,
		resourceARN, desiredTags, existingTags,
	)
}

// harnessAuthorizerConfigurationToSDK converts the CRD AuthorizerConfiguration
// union to the SDK union type. UpdateHarnessInput wraps this value in a PATCH
// wrapper (UpdatedAuthorizerConfiguration), so the standard generated update
// set is suppressed and the value is populated in the post-build-request hook.
func harnessAuthorizerConfigurationToSDK(
	c *svcapitypes.AuthorizerConfiguration,
) svcsdktypes.AuthorizerConfiguration {
	if c == nil {
		return nil
	}
	if c.CustomJWTAuthorizer != nil {
		parent := &svcsdktypes.AuthorizerConfigurationMemberCustomJWTAuthorizer{}
		jwt := svcsdktypes.CustomJWTAuthorizerConfiguration{}
		src := c.CustomJWTAuthorizer
		if src.AllowedAudience != nil {
			jwt.AllowedAudience = aws.ToStringSlice(src.AllowedAudience)
		}
		if src.AllowedClients != nil {
			jwt.AllowedClients = aws.ToStringSlice(src.AllowedClients)
		}
		if src.AllowedScopes != nil {
			jwt.AllowedScopes = aws.ToStringSlice(src.AllowedScopes)
		}
		if src.DiscoveryURL != nil {
			jwt.DiscoveryUrl = src.DiscoveryURL
		}
		if src.CustomClaims != nil {
			claims := make([]svcsdktypes.CustomClaimValidationType, 0, len(src.CustomClaims))
			for _, cc := range src.CustomClaims {
				if cc == nil {
					continue
				}
				claim := svcsdktypes.CustomClaimValidationType{}
				if cc.InboundTokenClaimName != nil {
					claim.InboundTokenClaimName = cc.InboundTokenClaimName
				}
				if cc.InboundTokenClaimValueType != nil {
					claim.InboundTokenClaimValueType = svcsdktypes.InboundTokenClaimValueType(*cc.InboundTokenClaimValueType)
				}
				if cc.AuthorizingClaimMatchValue != nil {
					mv := &svcsdktypes.AuthorizingClaimMatchValueType{}
					amv := cc.AuthorizingClaimMatchValue
					if amv.ClaimMatchOperator != nil {
						mv.ClaimMatchOperator = svcsdktypes.ClaimMatchOperatorType(*amv.ClaimMatchOperator)
					}
					if amv.ClaimMatchValue != nil {
						if amv.ClaimMatchValue.MatchValueString != nil {
							mv.ClaimMatchValue = &svcsdktypes.ClaimMatchValueTypeMemberMatchValueString{
								Value: *amv.ClaimMatchValue.MatchValueString,
							}
						} else if amv.ClaimMatchValue.MatchValueStringList != nil {
							mv.ClaimMatchValue = &svcsdktypes.ClaimMatchValueTypeMemberMatchValueStringList{
								Value: aws.ToStringSlice(amv.ClaimMatchValue.MatchValueStringList),
							}
						}
					}
					claim.AuthorizingClaimMatchValue = mv
				}
				claims = append(claims, claim)
			}
			jwt.CustomClaims = claims
		}
		parent.Value = jwt
		return parent
	}
	return nil
}

// harnessEnvironmentArtifactToSDK converts the CRD HarnessEnvironmentArtifact
// union to the SDK union type for use in the update PATCH wrapper.
func harnessEnvironmentArtifactToSDK(
	a *svcapitypes.HarnessEnvironmentArtifact,
) svcsdktypes.HarnessEnvironmentArtifact {
	if a == nil {
		return nil
	}
	if a.ContainerConfiguration != nil {
		parent := &svcsdktypes.HarnessEnvironmentArtifactMemberContainerConfiguration{}
		cc := svcsdktypes.ContainerConfiguration{}
		if a.ContainerConfiguration.ContainerURI != nil {
			cc.ContainerUri = a.ContainerConfiguration.ContainerURI
		}
		parent.Value = cc
		return parent
	}
	return nil
}

// harnessMemoryConfigurationToSDK converts the CRD HarnessMemoryConfiguration
// union to the SDK union type for use in the update PATCH wrapper.
func harnessMemoryConfigurationToSDK(
	m *svcapitypes.HarnessMemoryConfiguration,
) svcsdktypes.HarnessMemoryConfiguration {
	if m == nil {
		return nil
	}
	if m.AgentCoreMemoryConfiguration != nil {
		parent := &svcsdktypes.HarnessMemoryConfigurationMemberAgentCoreMemoryConfiguration{}
		src := m.AgentCoreMemoryConfiguration
		cfg := svcsdktypes.HarnessAgentCoreMemoryConfiguration{}
		if src.ActorID != nil {
			cfg.ActorId = src.ActorID
		}
		// The managed-memory Arn is server-populated and ignored on
		// Create/Update input, so it is not part of the CRD spec and is
		// not sent here.
		if src.MessagesCount != nil {
			v := int32(*src.MessagesCount)
			cfg.MessagesCount = &v
		}
		if src.RetrievalConfig != nil {
			rc := map[string]svcsdktypes.HarnessAgentCoreMemoryRetrievalConfig{}
			for k, val := range src.RetrievalConfig {
				if val == nil {
					continue
				}
				entry := svcsdktypes.HarnessAgentCoreMemoryRetrievalConfig{}
				if val.RelevanceScore != nil {
					s := float32(*val.RelevanceScore)
					entry.RelevanceScore = &s
				}
				if val.StrategyID != nil {
					entry.StrategyId = val.StrategyID
				}
				if val.TopK != nil {
					t := int32(*val.TopK)
					entry.TopK = &t
				}
				rc[k] = entry
			}
			cfg.RetrievalConfig = rc
		}
		parent.Value = cfg
		return parent
	}
	if m.Disabled != nil {
		return &svcsdktypes.HarnessMemoryConfigurationMemberDisabled{
			Value: svcsdktypes.HarnessDisabledMemoryConfiguration{},
		}
	}
	if m.ManagedMemoryConfiguration != nil {
		parent := &svcsdktypes.HarnessMemoryConfigurationMemberManagedMemoryConfiguration{}
		src := m.ManagedMemoryConfiguration
		cfg := svcsdktypes.HarnessManagedMemoryConfiguration{}
		// The managed-memory Arn is server-populated and ignored on
		// Create/Update input, so it is not part of the CRD spec and is
		// not sent here.
		if src.EncryptionKeyARN != nil {
			cfg.EncryptionKeyArn = src.EncryptionKeyARN
		}
		if src.EventExpiryDuration != nil {
			v := int32(*src.EventExpiryDuration)
			cfg.EventExpiryDuration = &v
		}
		if src.Strategies != nil {
			strategies := make([]svcsdktypes.HarnessManagedMemoryStrategyType, 0, len(src.Strategies))
			for _, s := range src.Strategies {
				if s == nil {
					continue
				}
				strategies = append(strategies, svcsdktypes.HarnessManagedMemoryStrategyType(*s))
			}
			cfg.Strategies = strategies
		}
		parent.Value = cfg
		return parent
	}
	return nil
}
