package provider

import (
	"reflect"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// TestVaultUserInput_AllFieldsValidated verifies that MatchesVault() checks ALL fields
// in VaultUserInput struct. This test uses reflection to automatically detect fields,
// so when you add a new field to VaultUserInput, this test will fail until you
// add validation for it in MatchesVault().
func TestVaultUserInput_AllFieldsValidated(t *testing.T) {
	// Use reflection to get all fields in VaultUserInput
	inputType := reflect.TypeOf(VaultUserInput{})
	numFields := inputType.NumField()

	t.Logf("VaultUserInput has %d fields", numFields)

	// Create base input and vault that match
	baseInput := VaultUserInput{
		Name:          "Test Vault",
		Region:        "us-east-1",
		CloudProvider: "AWS",
		AwsKmsKeyArn:  nil,
	}

	baseVault := &externalEonSdkAPI.BackupVault{
		Id:                "test-id",
		Name:              "Test Vault",
		Region:            "us-east-1",
		VaultAccountId:    "vault-123",
		ProviderAccountId: "provider-123",
		IsManagedByEon:    true,
	}
	awsProvider := externalEonSdkAPI.AWS
	baseVault.VaultAttributes = externalEonSdkAPI.VaultProviderAttributes{
		CloudProvider: awsProvider,
	}
	baseVault.VaultAttributes.SetAws(externalEonSdkAPI.AwsVaultConfig{})

	// Test each field individually
	for i := 0; i < numFields; i++ {
		field := inputType.Field(i)
		fieldName := field.Name

		t.Run("validates_"+fieldName, func(t *testing.T) {
			// Create a copy of base input and modify this field
			modifiedInput := baseInput
			modifiedVault := *baseVault

			switch fieldName {
			case "Name":
				modifiedInput.Name = "Different Name"
				matches, reason := modifiedInput.MatchesVault(&modifiedVault)
				if matches {
					t.Errorf("Field '%s': Expected mismatch to be detected, but MatchesVault returned true", fieldName)
				}
				t.Logf("Field '%s': ✓ Mismatch detected - %s", fieldName, reason)

			case "Region":
				modifiedInput.Region = "us-west-2"
				matches, reason := modifiedInput.MatchesVault(&modifiedVault)
				if matches {
					t.Errorf("Field '%s': Expected mismatch to be detected, but MatchesVault returned true", fieldName)
				}
				t.Logf("Field '%s': ✓ Mismatch detected - %s", fieldName, reason)

			case "CloudProvider":
				modifiedInput.CloudProvider = "AZURE"
				matches, reason := modifiedInput.MatchesVault(&modifiedVault)
				if matches {
					t.Errorf("Field '%s': Expected mismatch to be detected, but MatchesVault returned true", fieldName)
				}
				t.Logf("Field '%s': ✓ Mismatch detected - %s", fieldName, reason)

			case "AwsKmsKeyArn":
				existingKey := "arn:aws:kms:us-east-1:123456789012:key/existing"
				differentKey := "arn:aws:kms:us-east-1:123456789012:key/different"

				awsConfig := externalEonSdkAPI.AwsVaultConfig{}
				awsConfig.SetEncryptionKey(existingKey)
				modifiedVault.VaultAttributes.SetAws(awsConfig)

				modifiedInput.AwsKmsKeyArn = &differentKey

				matches, reason := modifiedInput.MatchesVault(&modifiedVault)
				if matches {
					t.Errorf("Field '%s': Expected mismatch to be detected, but MatchesVault returned true", fieldName)
				}
				t.Logf("Field '%s': ✓ Mismatch detected - %s", fieldName, reason)

			default:
				t.Errorf("⚠️  UNHANDLED FIELD: '%s'\n\n"+
					"A new field was added to VaultUserInput but this test doesn't know how to test it!\n\n"+
					"You MUST:\n"+
					"1. Add a test case for this field in the switch statement above\n"+
					"2. Verify MatchesVault() validates this field\n"+
					"3. If MatchesVault() doesn't validate it yet, add the validation logic",
					fieldName)
			}
		})
	}

	// Final sanity check: base input should match base vault
	t.Run("base_input_matches_base_vault", func(t *testing.T) {
		matches, reason := baseInput.MatchesVault(baseVault)
		if !matches {
			t.Errorf("Base input should match base vault, but got mismatch: %s", reason)
		}
		t.Log("✓ Base input correctly matches base vault")
	})
}
