package terraform

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aquasecurity/trivy/internal/testutil"
	"github.com/aquasecurity/trivy/pkg/iac/providers"
	"github.com/aquasecurity/trivy/pkg/iac/rego"
	"github.com/aquasecurity/trivy/pkg/iac/rules"
	"github.com/aquasecurity/trivy/pkg/iac/scan"
	"github.com/aquasecurity/trivy/pkg/iac/scanners/options"
	"github.com/aquasecurity/trivy/pkg/iac/severity"
	"github.com/aquasecurity/trivy/pkg/iac/state"
	"github.com/aquasecurity/trivy/pkg/iac/types"
)

const emptyBucketRule = `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: s3
#           provider: aws
package defsec.test.aws1
deny[res] {
  bucket := input.aws.s3.buckets[_]
  bucket.name.value == ""
  res := result.new("The name of the bucket must not be empty", bucket)
}
`

func Test_OptionWithPolicyDirs(t *testing.T) {

	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_s3_bucket" "my-bucket" {
	bucket = "evil"
}
`,
		"/rules/test.rego": `
package defsec.abcdefg

__rego_metadata__ := {
	"id": "TEST123",
	"avd_id": "AVD-TEST-0123",
	"title": "Buckets should not be evil",
	"short_code": "no-evil-buckets",
	"severity": "CRITICAL",
	"type": "DefSec Security Check",
	"description": "You should not allow buckets to be evil",
	"recommended_actions": "Use a good bucket instead",
	"url": "https://google.com/search?q=is+my+bucket+evil",
}

__rego_input__ := {
	"combine": false,
	"selector": [{"type": "defsec", "subtypes": [{"service": "s3", "provider": "aws"}]}],
}

deny[cause] {
	bucket := input.aws.s3.buckets[_]
	bucket.name.value == "evil"
	cause := bucket.name
}
`,
	})

	scanner := New(
		rego.WithPolicyFilesystem(fs),
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)

	failure := results.GetFailed()[0]

	assert.Equal(t, "AVD-TEST-0123", failure.Rule().AVDID)

	actualCode, err := failure.GetCode()
	require.NoError(t, err)
	for i := range actualCode.Lines {
		actualCode.Lines[i].Highlighted = ""
	}
	assert.Equal(t, []scan.Line{
		{
			Number:     2,
			Content:    "resource \"aws_s3_bucket\" \"my-bucket\" {",
			IsCause:    false,
			FirstCause: false,
			LastCause:  false,
			Annotation: "",
		},
		{
			Number:     3,
			Content:    "\tbucket = \"evil\"",
			IsCause:    true,
			FirstCause: true,
			LastCause:  true,
			Annotation: "",
		},
		{
			Number:     4,
			Content:    "}",
			IsCause:    false,
			FirstCause: false,
			LastCause:  false,
			Annotation: "",
		},
	}, actualCode.Lines)

}

func Test_OptionWithPolicyNamespaces(t *testing.T) {

	tests := []struct {
		includedNamespaces []string
		policyNamespace    string
		wantFailure        bool
	}{
		{
			includedNamespaces: nil,
			policyNamespace:    "blah",
			wantFailure:        false,
		},
		{
			includedNamespaces: nil,
			policyNamespace:    "appshield.something",
			wantFailure:        true,
		},
		{
			includedNamespaces: nil,
			policyNamespace:    "defsec.blah",
			wantFailure:        true,
		},
		{
			includedNamespaces: []string{"user"},
			policyNamespace:    "users",
			wantFailure:        false,
		},
		{
			includedNamespaces: []string{"users"},
			policyNamespace:    "something.users",
			wantFailure:        false,
		},
		{
			includedNamespaces: []string{"users"},
			policyNamespace:    "users",
			wantFailure:        true,
		},
		{
			includedNamespaces: []string{"users"},
			policyNamespace:    "users.my_rule",
			wantFailure:        true,
		},
		{
			includedNamespaces: []string{
				"a",
				"users",
				"b",
			},
			policyNamespace: "users",
			wantFailure:     true,
		},
		{
			includedNamespaces: []string{"user"},
			policyNamespace:    "defsec",
			wantFailure:        true,
		},
	}

	for i, test := range tests {

		t.Run(strconv.Itoa(i), func(t *testing.T) {

			fs := testutil.CreateFS(t, map[string]string{
				"/code/main.tf": `
resource "aws_s3_bucket" "my-bucket" {
	bucket = "evil"
}
`,
				"/rules/test.rego": fmt.Sprintf(`
# METADATA
# custom:
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#       - service: s3
#         provider: aws
package %s

deny[cause] {
bucket := input.aws.s3.buckets[_]
bucket.name.value == "evil"
cause := bucket.name
}

				`, test.policyNamespace),
			})

			scanner := New(
				rego.WithPolicyDirs("rules"),
				rego.WithPolicyNamespaces(test.includedNamespaces...),
			)

			results, err := scanner.ScanFS(context.TODO(), fs, "code")
			require.NoError(t, err)

			var found bool
			for _, result := range results.GetFailed() {
				if result.RegoNamespace() == test.policyNamespace && result.RegoRule() == "deny" {
					found = true
					break
				}
			}
			assert.Equal(t, test.wantFailure, found)
		})
	}

}

func Test_OptionWithRegoOnly(t *testing.T) {

	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_s3_bucket" "my-bucket" {
	bucket = "evil"
}
`,
		"/rules/test.rego": `
package defsec.abcdefg

__rego_metadata__ := {
	"id": "TEST123",
	"avd_id": "AVD-TEST-0123",
	"title": "Buckets should not be evil",
	"short_code": "no-evil-buckets",
	"severity": "CRITICAL",
	"type": "DefSec Security Check",
	"description": "You should not allow buckets to be evil",
	"recommended_actions": "Use a good bucket instead",
	"url": "https://google.com/search?q=is+my+bucket+evil",
}

__rego_input__ := {
	"combine": false,
	"selector": [{"type": "defsec", "subtypes": [{"service": "s3", "provider": "aws"}]}],
}

deny[cause] {
	bucket := input.aws.s3.buckets[_]
	bucket.name.value == "evil"
	cause := bucket.name
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-TEST-0123", results[0].Rule().AVDID)
}

func Test_OptionWithRegoOnly_CodeHighlighting(t *testing.T) {

	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_s3_bucket" "my-bucket" {
	bucket = "evil"
}
`,
		"/rules/test.rego": `
package defsec.abcdefg

__rego_metadata__ := {
	"id": "TEST123",
	"avd_id": "AVD-TEST-0123",
	"title": "Buckets should not be evil",
	"short_code": "no-evil-buckets",
	"severity": "CRITICAL",
	"type": "DefSec Security Check",
	"description": "You should not allow buckets to be evil",
	"recommended_actions": "Use a good bucket instead",
	"url": "https://google.com/search?q=is+my+bucket+evil",
}

__rego_input__ := {
	"combine": false,
	"selector": [{"type": "defsec", "subtypes": [{"service": "s3", "provider": "aws"}]}],
}

deny[res] {
	bucket := input.aws.s3.buckets[_]
	bucket.name.value == "evil"
	res := result.new("oh no", bucket.name)
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-TEST-0123", results[0].Rule().AVDID)
	assert.NotNil(t, results[0].Metadata().Range().GetFS())
}

func Test_IAMPolicyRego(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_sqs_queue_policy" "bad_example" {
   queue_url = aws_sqs_queue.q.id

   policy = <<POLICY
 {
   "Statement": [
     {
       "Effect": "Allow",
       "Principal": "*",
       "Action": "*"
     }
   ]
 }
 POLICY
 }`,
		"/rules/test.rego": `
# METADATA
# title: Buckets should not be evil
# description: You should not allow buckets to be evil
# scope: package
# schemas:
#  - input: schema.input
# related_resources:
# - https://google.com/search?q=is+my+bucket+evil
# custom:
#   id: TEST123
#   avd_id: AVD-TEST-0123
#   short_code: no-evil-buckets
#   severity: CRITICAL
#   recommended_action: Use a good bucket instead
#   input:
#     selector:
#     - type: cloud
#       subtypes: 
#         - service: sqs
#           provider: aws
package defsec.abcdefg


deny[res] {
	queue := input.aws.sqs.queues[_]
	policy := queue.policies[_]
	doc := json.unmarshal(policy.document.value)
	statement = doc.Statement[_]
	action := statement.Action[_]
	action == "*"
	res := result.new("SQS Policy contains wildcard in action", policy.document)
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-TEST-0123", results[0].Rule().AVDID)
	assert.NotNil(t, results[0].Metadata().Range().GetFS())

}

func Test_ContainerDefinitionRego(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_ecs_task_definition" "test" {
  family                = "test"
  container_definitions = <<TASK_DEFINITION
[
  {
	"privileged": true,
    "cpu": 10,
    "command": ["sleep", "10"],
    "entryPoint": ["/"],
    "environment": [
      {"name": "VARNAME", "value": "VARVAL"}
    ],
    "essential": true,
    "image": "jenkins",
    "memory": 128,
    "name": "jenkins",
    "portMappings": [
      {
        "containerPort": 80,
        "hostPort": 8080
      }
    ],
        "resourceRequirements":[
            {
                "type":"InferenceAccelerator",
                "value":"device_1"
            }
        ]
  }
]
TASK_DEFINITION

  inference_accelerator {
    device_name = "device_1"
    device_type = "eia1.medium"
  }
}`,
		"/rules/test.rego": `
package defsec.abcdefg


__rego_metadata__ := {
	"id": "TEST123",
	"avd_id": "AVD-TEST-0123",
	"title": "Buckets should not be evil",
	"short_code": "no-evil-buckets",
	"severity": "CRITICAL",
	"type": "DefSec Security Check",
	"description": "You should not allow buckets to be evil",
	"recommended_actions": "Use a good bucket instead",
	"url": "https://google.com/search?q=is+my+bucket+evil",
}

__rego_input__ := {
	"combine": false,
	"selector": [{"type": "defsec", "subtypes": [{"service": "ecs", "provider": "aws"}]}],
}

deny[res] {
	definition := input.aws.ecs.taskdefinitions[_].containerdefinitions[_]
	definition.privileged.value == true
	res := result.new("Privileged container detected", definition.privileged)
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-TEST-0123", results[0].Rule().AVDID)
	assert.NotNil(t, results[0].Metadata().Range().GetFS())

}

func Test_S3_Linking(t *testing.T) {

	code := `
## tfsec:ignore:aws-s3-enable-bucket-encryption
## tfsec:ignore:aws-s3-enable-bucket-logging
## tfsec:ignore:aws-s3-enable-versioning
resource "aws_s3_bucket" "blubb" {
  bucket = "test"
}

resource "aws_s3_bucket_public_access_block" "audit_logs_athena" {
  bucket = aws_s3_bucket.blubb.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# tfsec:ignore:aws-s3-enable-bucket-encryption
# tfsec:ignore:aws-s3-enable-bucket-logging
# tfsec:ignore:aws-s3-enable-versioning
resource "aws_s3_bucket" "foo" {
  bucket        = "prefix-" # remove this variable and it works; does not report
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "foo" {
  bucket = aws_s3_bucket.foo.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

`

	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": code,
	})

	scanner := New()

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	failed := results.GetFailed()
	for _, result := range failed {
		// public access block
		assert.NotEqual(t, "AVD-AWS-0094", result.Rule().AVDID, "AVD-AWS-0094 should not be reported - was found at "+result.Metadata().Range().String())
		// encryption
		assert.NotEqual(t, "AVD-AWS-0088", result.Rule().AVDID)
		// logging
		assert.NotEqual(t, "AVD-AWS-0089", result.Rule().AVDID)
		// versioning
		assert.NotEqual(t, "AVD-AWS-0090", result.Rule().AVDID)
	}
}

func Test_S3_Linking_PublicAccess(t *testing.T) {

	code := `
resource "aws_s3_bucket" "testA" {
  bucket = "com.test.testA"
}

resource "aws_s3_bucket_acl" "testA" {
  bucket = aws_s3_bucket.testA.id
  acl    = "private"
}

resource "aws_s3_bucket_public_access_block" "testA" {
  bucket = aws_s3_bucket.testA.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket" "testB" {
  bucket = "com.test.testB"
}

resource "aws_s3_bucket_acl" "testB" {
  bucket = aws_s3_bucket.testB.id
  acl    = "private"
}

resource "aws_s3_bucket_public_access_block" "testB" {
  bucket = aws_s3_bucket.testB.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

`

	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": code,
	})

	scanner := New()

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	for _, result := range results.GetFailed() {
		// public access block
		assert.NotEqual(t, "AVD-AWS-0094", result.Rule().AVDID)
	}

}

// PoC for replacing Go with Rego: AVD-AWS-0001
func Test_RegoRules(t *testing.T) {

	fs := testutil.CreateFS(t, map[string]string{
		"/code/main.tf": `
resource "aws_apigatewayv2_stage" "bad_example" {
  api_id = aws_apigatewayv2_api.example.id
  name   = "example-stage"
}
`,
		"/rules/test.rego": `# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: apigateway
#           provider: aws
package builtin.cloud.AWS0001

deny[res] {
	api := input.aws.apigateway.v1.apis[_]
	stage := api.stages[_]
	isManaged(stage)
	stage.accesslogging.cloudwatchloggrouparn.value == ""
	res := result.new("Access logging is not configured.", stage.accesslogging.cloudwatchloggrouparn)
}

deny[res] {
	api := input.aws.apigateway.v2.apis[_]
	stage := api.stages[_]
	isManaged(stage)
	stage.accesslogging.cloudwatchloggrouparn.value == ""
	res := result.new("Access logging is not configured.", stage.accesslogging.cloudwatchloggrouparn)
}
`,
	})

	scanner := New(
		rego.WithPolicyFilesystem(fs),
		rego.WithPolicyDirs("rules"),
		options.ScannerWithRegoOnly(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results.GetFailed(), 1)

	failure := results.GetFailed()[0]

	assert.Equal(t, "AVD-AWS-0001", failure.Rule().AVDID)

	actualCode, err := failure.GetCode()
	require.NoError(t, err)
	for i := range actualCode.Lines {
		actualCode.Lines[i].Highlighted = ""
	}
	assert.Equal(t, []scan.Line{
		{
			Number:     2,
			Content:    "resource \"aws_apigatewayv2_stage\" \"bad_example\" {",
			IsCause:    true,
			FirstCause: true,
			LastCause:  false,
			Annotation: "",
		},
		{
			Number:     3,
			Content:    "  api_id = aws_apigatewayv2_api.example.id",
			IsCause:    true,
			FirstCause: false,
			LastCause:  false,
			Annotation: "",
		},
		{
			Number:     4,
			Content:    "  name   = \"example-stage\"",
			IsCause:    true,
			FirstCause: false,
			LastCause:  false,
			Annotation: "",
		},
		{
			Number:     5,
			Content:    "}",
			IsCause:    true,
			FirstCause: false,
			LastCause:  true,
			Annotation: "",
		},
	}, actualCode.Lines)
}

func Test_OptionWithConfigsFileSystem(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": `
variable "bucket_name" {
  type = string
}
resource "aws_s3_bucket" "main" {
  bucket = var.bucket_name
}
`,
		"rules/bucket_name.rego": emptyBucketRule,
	})

	configsFS := testutil.CreateFS(t, map[string]string{
		"main.tfvars": `
bucket_name = "test"
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		rego.WithPolicyFilesystem(fs),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		ScannerWithAllDirectories(true),
		ScannerWithTFVarsPaths("main.tfvars"),
		ScannerWithConfigsFileSystem(configsFS),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Len(t, results.GetPassed(), 1)
}

func Test_OptionWithConfigsFileSystem_ConfigInCode(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": `
variable "bucket_name" {
  type = string
}
resource "aws_s3_bucket" "main" {
  bucket = var.bucket_name
}
`,
		"rules/bucket_name.rego": emptyBucketRule,
		"main.tfvars": `
bucket_name = "test"
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		rego.WithPolicyFilesystem(fs),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		ScannerWithAllDirectories(true),
		ScannerWithTFVarsPaths("main.tfvars"),
		ScannerWithConfigsFileSystem(fs),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Len(t, results.GetPassed(), 1)
}

func Test_DoNotScanNonRootModules(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"/code/app1/main.tf": `
module "s3" {
  source      = "./modules/s3"
  bucket_name = "test"
}
`,
		"/code/app1/modules/s3/main.tf": `
variable "bucket_name" {
  type = string
}

resource "aws_s3_bucket" "main" {
  bucket = var.bucket_name
}
`,
		"/code/app1/app2/main.tf": `
module "s3" {
  source      = "../modules/s3"
  bucket_name = "test"
}

module "ec2" {
  source = "./modules/ec2"
}
`,
		"/code/app1/app2/modules/ec2/main.tf": `
variable "security_group_description" {
	type = string
}
resource "aws_security_group" "main" {
	description = var.security_group_description
}
`,
		"/rules/bucket_name.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: s3
#           provider: aws
package defsec.test.aws1
deny[res] {
  bucket := input.aws.s3.buckets[_]
  bucket.name.value == ""
  res := result.new("The name of the bucket must not be empty", bucket)
}
`,
		"/rules/sec_group_description.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0002
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: ec2
#           provider: aws
package defsec.test.aws2
deny[res] {
  group := input.aws.ec2.securitygroups[_]
  group.description.value == ""
  res := result.new("The description of the security group must not be empty", group)
}
`,
	})

	scanner := New(
		rego.WithPolicyFilesystem(fs),
		rego.WithPolicyDirs("rules"),
		rego.WithEmbeddedPolicies(false),
		rego.WithEmbeddedLibraries(false),
		options.ScannerWithRegoOnly(true),
		ScannerWithAllDirectories(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	assert.Len(t, results.GetPassed(), 2)
	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-AWS-0002", results.GetFailed()[0].Rule().AVDID)
}

func Test_RoleRefToOutput(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": `
module "this" {
  source = "./modules/iam"
}

resource "aws_iam_role_policy" "bad-policy" {
  name     = "bad-policy"
  role     = module.this.role_name
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect   = "Allow"
        Action   = "*"
        Resource = "*"
      },
    ]
  })
}
		`,
		"code/modules/iam/main.tf": `
resource "aws_iam_role" "example" {
  name               = "example"
  assume_role_policy = jsonencode({})
}

output "role_name" {
  value = aws_iam_role.example.id
}
		`,
		"rules/test.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: iam
#           provider: aws
package defsec.test.aws1
deny[res] {
  policy := input.aws.iam.roles[_].policies[_]
  policy.name.value == "bad-policy"
  res := result.new("Deny!", policy)
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		rego.WithPolicyFilesystem(fs),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		ScannerWithAllDirectories(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Len(t, results.GetFailed(), 1)
}

func Test_RegoRefToAwsProviderAttributes(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"code/providers.tf": `
provider "aws" {
  region  = "us-east-2"
  default_tags {
    tags = {
      Environment = "Local"
      Name        = "LocalStack"
    }
  }
}
`,
		"rules/region.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: meta
#           provider: aws
package defsec.test.aws1
deny[res] {
  region := input.aws.meta.tfproviders[_].region
  region.value != "us-east-1"
  res := result.new("Only the 'us-east-1' region is allowed!", region)
}
`,
		"rules/tags.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0002
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: meta
#           provider: aws
package defsec.test.aws2
deny[res] {
  provider := input.aws.meta.tfproviders[_]
  tags = provider.defaulttags.tags.value
  not tags.Environment
  res := result.new("provider should have the following default tags: 'Environment'", tags)
}`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		rego.WithPolicyFilesystem(fs),
		options.ScannerWithRegoOnly(true),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		ScannerWithAllDirectories(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results, 2)

	require.Len(t, results.GetFailed(), 1)
	assert.Equal(t, "AVD-AWS-0001", results.GetFailed()[0].Rule().AVDID)

	require.Len(t, results.GetPassed(), 1)
	assert.Equal(t, "AVD-AWS-0002", results.GetPassed()[0].Rule().AVDID)
}

func TestScanModuleWithCount(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"code/main.tf": `
module "this" {
  count = 0
  source = "./modules/s3"
}`,
		"code/modules/s3/main.tf": `
module "this" {
  source = "./modules/logging"
}
resource "aws_s3_bucket" "this" {
  bucket = "test"
}`,
		"code/modules/s3/modules/logging/main.tf": `
resource "aws_s3_bucket" "this" {
  bucket = "test1"
}`,
		"code/example/main.tf": `
module "this" {
  source = "../modules/s3"
}`,
		"rules/region.rego": `
# METADATA
# schemas:
# - input: schema.input
# custom:
#   avd_id: AVD-AWS-0001
#   input:
#     selector:
#     - type: cloud
#       subtypes:
#         - service: s3
#           provider: aws
package user.test.aws1
deny[res] {
	bucket := input.aws.s3.buckets[_]
	bucket.name.value == "test"
  res := result.new("bucket with test name is not allowed!", bucket)
}
`,
	})

	scanner := New(
		rego.WithPolicyDirs("rules"),
		rego.WithPolicyFilesystem(fs),
		options.ScannerWithRegoOnly(true),
		rego.WithPolicyNamespaces("user"),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		rego.WithRegoErrorLimits(0),
		ScannerWithAllDirectories(true),
	)

	results, err := scanner.ScanFS(context.TODO(), fs, "code")
	require.NoError(t, err)

	require.Len(t, results, 1)

	failed := results.GetFailed()

	assert.Len(t, failed, 1)

	occurrences := failed[0].Occurrences()
	assert.Equal(t, "code/example/main.tf", occurrences[0].Filename)
}

func TestSkipDeprecatedGoChecks(t *testing.T) {

	check := scan.Rule{
		Provider:  providers.AWSProvider,
		Service:   "service",
		ShortCode: "abc",
		Severity:  severity.High,
		Check: func(s *state.State) (results scan.Results) {
			results.Add("Deny", types.NewTestMetadata())
			return
		},
	}

	fsys := testutil.CreateFS(t, map[string]string{
		"main.tf": `resource "foo" "bar" {}`,
	})

	scanner := New(
		rego.WithPolicyFilesystem(fsys),
		rego.WithEmbeddedLibraries(false),
		rego.WithEmbeddedPolicies(false),
		ScannerWithAllDirectories(true),
	)

	t.Run("deprecated", func(t *testing.T) {
		check.Deprecated = true
		reg := rules.Register(check)
		defer rules.Deregister(reg)

		results, err := scanner.ScanFS(context.TODO(), fsys, ".")
		require.NoError(t, err)

		require.Empty(t, results)
	})

	t.Run("not deprecated", func(t *testing.T) {
		check.Deprecated = false
		reg := rules.Register(check)
		defer rules.Deregister(reg)

		results, err := scanner.ScanFS(context.TODO(), fsys, ".")
		require.NoError(t, err)

		require.Len(t, results, 1)
	})
}

func TestSkipDir(t *testing.T) {
	fs := testutil.CreateFS(t, map[string]string{
		"deployments/main.tf": `
module "use_bad_configuration" {
  source = "../modules"
}

module "use_bad_configuration_2" {
  source = "../modules/modules2"
}
`,
		"modules/misconfig.tf": `data "aws_iam_policy_document" "bad" {
  statement {
    actions = [
      "apigateway:*",
    ]

    resources = [
      "*",
    ]
  }
}

resource "aws_iam_policy" "bad_configuration" {
  name_prefix = local.setup_role_name
  policy      = data.aws_iam_policy_document.bad.json
}
`,
		"modules/modules2/misconfig.tf": `data "aws_iam_policy_document" "bad" {
  statement {
    actions = [
      "apigateway:*",
    ]

    resources = [
      "*",
    ]
  }
}

resource "aws_iam_policy" "bad_configuration" {
  name_prefix = local.setup_role_name
  policy      = data.aws_iam_policy_document.bad.json
}
`,
	})

	t.Run("use skip-dir option", func(t *testing.T) {
		scanner := New(
			options.ScannerWithIncludeDeprecatedChecks(true),
			ScannerWithSkipDirs([]string{"**/modules/**"}),
			ScannerWithAllDirectories(true),
		)

		results, err := scanner.ScanFS(context.TODO(), fs, "deployments")
		require.NoError(t, err)

		assert.Empty(t, results)
	})

	t.Run("use skip-files option", func(t *testing.T) {
		scanner := New(
			options.ScannerWithIncludeDeprecatedChecks(true),
			ScannerWithSkipFiles([]string{"**/modules/**/*.tf"}),
			ScannerWithAllDirectories(true),
		)

		results, err := scanner.ScanFS(context.TODO(), fs, "deployments")
		require.NoError(t, err)

		assert.Empty(t, results)
	})

	t.Run("non existing value for skip-files option", func(t *testing.T) {
		scanner := New(
			options.ScannerWithIncludeDeprecatedChecks(true),
			ScannerWithSkipFiles([]string{"foo/bar*.tf"}),
			ScannerWithAllDirectories(true),
		)

		results, err := scanner.ScanFS(context.TODO(), fs, "deployments")
		require.NoError(t, err)

		assert.Len(t, results, 4)
	})

	t.Run("empty skip-files option", func(t *testing.T) {
		scanner := New(
			options.ScannerWithIncludeDeprecatedChecks(true),
			ScannerWithAllDirectories(true),
		)

		results, err := scanner.ScanFS(context.TODO(), fs, "deployments")
		require.NoError(t, err)

		assert.Len(t, results, 4)
	})
}
