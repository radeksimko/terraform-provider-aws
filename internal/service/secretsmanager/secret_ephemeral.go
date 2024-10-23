// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package secretsmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @EphemeralResource(aws_secretsmanager_secret, name="Secret")
func newEphemeralSecret(_ context.Context) (ephemeral.EphemeralResourceWithConfigure, error) {
	return &ephemeralSecret{}, nil
}

type ephemeralSecret struct {
	framework.EphemeralResourceWithConfigure
}

func (e *ephemeralSecret) Metadata(_ context.Context, _ ephemeral.MetadataRequest, response *ephemeral.MetadataResponse) {
	response.TypeName = "aws_secretsmanager_secret"
}

func (e *ephemeralSecret) Schema(ctx context.Context, _ ephemeral.SchemaRequest, response *ephemeral.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"secret_id": schema.StringAttribute{
				Required: true,
			},
			"version_id": schema.StringAttribute{
				Optional: true,
			},
			"version_stage": schema.StringAttribute{
				Optional: true,
			},

			names.AttrARN: schema.StringAttribute{
				Computed: true,
			},
			names.AttrCreatedDate: schema.StringAttribute{
				Computed: true,
			},
			"secret_binary": schema.StringAttribute{
				Sensitive: true,
				Computed:  true,
			},
			"secret_string": schema.StringAttribute{
				Sensitive: true,
				Computed:  true,
			},
		},
	}
}

func (e *ephemeralSecret) Open(ctx context.Context, request ephemeral.OpenRequest, response *ephemeral.OpenResponse) {
	var data epSecretData
	conn := e.Meta().SecretsManagerClient(ctx)

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	var version string
	input := &secretsmanager.GetSecretValueInput{
		SecretId: data.SecretID.ValueStringPointer(),
	}

	if !data.VersionID.IsNull() {
		input.VersionId = data.VersionID.ValueStringPointer()
		version = data.VersionID.String()
	} else if !data.VersionStage.IsNull() {
		input.VersionStage = data.VersionStage.ValueStringPointer()
		version = data.VersionStage.String()
	}

	id := secretVersionCreateResourceID(data.SecretID.String(), version)
	output, err := findSecretVersion(ctx, conn, input)

	if err != nil {
		response.Diagnostics.AddError(
			fmt.Sprintf("failed reading Secrets Manager Secret Version (%s)", id),
			err.Error(),
		)
		return
	}

	data.ARN = fwflex.StringValueToFramework(ctx, *output.ARN)
	data.CreatedDate = fwflex.StringValueToFramework(ctx, output.CreatedDate.Format(time.RFC3339))
	data.SecretBinary = fwflex.StringValueToFramework(ctx, string(output.SecretBinary))
	data.SecretString = fwflex.StringValueToFramework(ctx, *output.SecretString)

	response.Diagnostics.Append(response.Result.Set(ctx, &data)...)
}

type epSecretData struct {
	SecretID     types.String `tfsdk:"secret_id"`
	VersionID    types.String `tfsdk:"version_id"`
	VersionStage types.String `tfsdk:"version_stage"`

	ARN          types.String `tfsdk:"arn"`
	CreatedDate  types.String `tfsdk:"created_date"`
	SecretBinary types.String `tfsdk:"secret_binary"`
	SecretString types.String `tfsdk:"secret_string"`
}
