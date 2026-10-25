package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

func executeRestorePickerRequest[T any](c *EonClient, baseError string, execute func() (*T, *http.Response, error)) (*T, error) {
	if err := c.tokenRefresher.EnsureValidToken(); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	response, httpResp, err := execute()
	if apiErr := c.handleAPIError(err, httpResp, baseError); apiErr != nil {
		return nil, apiErr
	}
	if httpResp == nil {
		return nil, fmt.Errorf("%s: received no HTTP response", baseError)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, &APIError{
			StatusCode: httpResp.StatusCode,
			Message:    string(body),
		}
	}

	return response, nil
}

// ListRestoreAwsVpcs lists VPCs in a restore account and region.
func (c *EonClient) ListRestoreAwsVpcs(ctx context.Context, accountID, region string) ([]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner, error) {
	response, err := executeRestorePickerRequest(c, "failed to list restore AWS VPCs", func() (*externalEonSdkAPI.ListAwsVpcsResponse, *http.Response, error) {
		return c.client.ListAwsVpcsAPI.ListAwsVpcs(ctx, c.projectID, accountID, region).Execute()
	})
	if err != nil {
		return nil, err
	}

	vpcs := response.GetVpcs()
	if vpcs == nil {
		return []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{}, nil
	}
	return vpcs, nil
}

// ListRestoreAwsSubnets lists subnets and their VPCs in a restore account and region.
func (c *EonClient) ListRestoreAwsSubnets(ctx context.Context, accountID, region string) ([]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner, error) {
	response, err := executeRestorePickerRequest(c, "failed to list restore AWS subnets", func() (*externalEonSdkAPI.ListSubnetsResponseV2, *http.Response, error) {
		return c.client.ListAwsSubnetsV2API.ListAwsSubnetsV2(ctx, c.projectID, accountID, region).Execute()
	})
	if err != nil {
		return nil, err
	}

	subnets := response.GetSubnets()
	if subnets == nil {
		return []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{}, nil
	}
	return subnets, nil
}

// ListRestoreAwsSecurityGroups lists security groups for an optional VPC and subnet.
func (c *EonClient) ListRestoreAwsSecurityGroups(ctx context.Context, accountID, region string, vpcID, subnetID *string) ([]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner, error) {
	request := c.client.ListAwsSecurityGroupsAPI.ListAwsSecurityGroups(ctx, c.projectID, accountID, region)
	if vpcID != nil {
		request = request.VpcId(*vpcID)
	}
	if subnetID != nil {
		request = request.SubnetId(*subnetID)
	}

	response, err := executeRestorePickerRequest(c, "failed to list restore AWS security groups", request.Execute)
	if err != nil {
		return nil, err
	}

	securityGroups := response.GetSecurityGroups()
	if securityGroups == nil {
		return []externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{}, nil
	}
	return securityGroups, nil
}

// ListRestoreAwsInstanceTypes lists restore instance types for a required subnet.
func (c *EonClient) ListRestoreAwsInstanceTypes(ctx context.Context, accountID, region, subnetID string, volumeCount *int32, totalVolumeBytes *int64) ([]string, error) {
	request := c.client.ListAwsInstanceTypesAPI.ListAwsInstanceTypes(ctx, c.projectID, accountID, region).SubnetId(subnetID)
	if volumeCount != nil {
		request = request.VolumeCount(*volumeCount)
	}
	if totalVolumeBytes != nil {
		request = request.TotalVolumeBytes(*totalVolumeBytes)
	}

	response, err := executeRestorePickerRequest(c, "failed to list restore AWS instance types", request.Execute)
	if err != nil {
		return nil, err
	}

	instanceTypes := response.GetInstanceTypes()
	if instanceTypes == nil {
		return []string{}, nil
	}
	return instanceTypes, nil
}

// ListRestoreEncryptionKeys lists encryption keys in a restore account and region.
func (c *EonClient) ListRestoreEncryptionKeys(ctx context.Context, accountID, region string, includeAwsManagedKeys *bool) ([]externalEonSdkAPI.EncryptionKey, error) {
	request := c.client.ListEncryptionKeysAPI.ListEncryptionKeys(ctx, c.projectID, accountID, region)
	if includeAwsManagedKeys != nil {
		request = request.IncludeAwsManagedKeys(*includeAwsManagedKeys)
	}

	response, err := executeRestorePickerRequest(c, "failed to list restore encryption keys", request.Execute)
	if err != nil {
		return nil, err
	}

	keys := response.GetKeys()
	if keys == nil {
		return []externalEonSdkAPI.EncryptionKey{}, nil
	}
	return keys, nil
}

// ListRestoreS3Buckets lists S3 buckets in a restore account.
func (c *EonClient) ListRestoreS3Buckets(ctx context.Context, accountID string) ([]externalEonSdkAPI.S3Bucket, error) {
	response, err := executeRestorePickerRequest(c, "failed to list restore S3 buckets", func() (*externalEonSdkAPI.S3Buckets, *http.Response, error) {
		return c.client.ListS3BucketsAPI.ListS3Buckets(ctx, c.projectID, accountID).Execute()
	})
	if err != nil {
		return nil, err
	}

	buckets := response.GetBuckets()
	if buckets == nil {
		return []externalEonSdkAPI.S3Bucket{}, nil
	}
	return buckets, nil
}
