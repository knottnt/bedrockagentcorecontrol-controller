	// AuthorizerConfiguration, EnvironmentArtifact and Memory use PATCH-wrapper
	// types in UpdateHarnessInput (Updated*), which differ from the bare union
	// types in CreateHarnessInput. The generated update-set for these fields is
	// suppressed in generator.yaml; populate them here by wrapping the converted
	// spec union value in its Updated* wrapper via the OptionalValue member.
	if delta.DifferentAt("Spec.AuthorizerConfiguration") {
		input.AuthorizerConfiguration = &svcsdktypes.UpdatedAuthorizerConfiguration{
			OptionalValue: harnessAuthorizerConfigurationToSDK(desired.ko.Spec.AuthorizerConfiguration),
		}
	}
	if delta.DifferentAt("Spec.EnvironmentArtifact") {
		input.EnvironmentArtifact = &svcsdktypes.UpdatedHarnessEnvironmentArtifact{
			OptionalValue: harnessEnvironmentArtifactToSDK(desired.ko.Spec.EnvironmentArtifact),
		}
	}
	if delta.DifferentAt("Spec.Memory") {
		input.Memory = &svcsdktypes.UpdatedHarnessMemoryConfiguration{
			OptionalValue: harnessMemoryConfigurationToSDK(desired.ko.Spec.Memory),
		}
	}
