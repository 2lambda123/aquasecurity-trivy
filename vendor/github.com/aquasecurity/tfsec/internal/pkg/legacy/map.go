package legacy

func FindID(longID string) string {
	return InvertedIDs[longID]
}

var InvertedIDs map[string]string

func init() {
	InvertedIDs = make(map[string]string)
	for legacy, modern := range IDs {
		InvertedIDs[modern] = legacy
	}
}

var IDs = map[string]string{
	"AWS061": "aws-api-gateway-enable-access-logging",
	"AWS025": "aws-api-gateway-use-secure-tls-policy",
	"AWS059": "aws-athena-enable-at-rest-encryption",
	"AWS060": "aws-athena-no-encryption-override",
	"AWS014": "aws-autoscaling-enable-at-rest-encryption",
	"AWS012": "aws-autoscaling-no-public-ip",
	"AWS071": "aws-cloudfront-enable-logging",
	"AWS045": "aws-cloudfront-enable-waf",
	"AWS020": "aws-cloudfront-enforce-https",
	"AWS021": "aws-cloudfront-use-secure-tls-policy",
	"AWS063": "aws-cloudtrail-enable-all-regions",
	"AWS065": "aws-cloudtrail-enable-at-rest-encryption",
	"AWS064": "aws-cloudtrail-enable-log-validation",
	"AWS089": "aws-cloudwatch-log-group-customer-key",
	"AWS080": "aws-codebuild-enable-encryption",
	"AWS085": "aws-config-aggregate-all-regions",
	"AWS081": "aws-dynamodb-enable-at-rest-encryption",
	"AWS086": "aws-dynamodb-enable-recovery",
	"AWS092": "aws-dynamodb-table-customer-key",
	"AWS079": "aws-ec2-enforce-http-token-imds",
	"AWS062": "aws-ec2-no-secrets-in-user-data",
	"AWS023": "aws-ecr-enable-image-scans",
	"AWS078": "aws-ecr-enforce-immutable-repository",
	"AWS093": "aws-ecr-repository-customer-key",
	"AWS090": "aws-ecs-enable-container-insight",
	"AWS096": "aws-ecs-enable-in-transit-encryption",
	"AWS013": "aws-ecs-no-plaintext-secrets",
	"AWS048": "aws-efs-enable-at-rest-encryption",
	"AWS067": "aws-eks-enable-control-plane-logging",
	"AWS066": "aws-eks-encrypt-secrets",
	"AWS069": "aws-eks-no-public-cluster-access",
	"AWS068": "aws-eks-no-public-cluster-access-to-cidr",
	"AWS031": "aws-elastic-search-enable-domain-encryption",
	"AWS057": "aws-elastic-search-enable-domain-logging",
	"AWS032": "aws-elastic-search-enable-in-transit-encryption",
	"AWS033": "aws-elastic-search-enforce-https",
	"AWS034": "aws-elastic-search-use-secure-tls-policy",
	"AWS035": "aws-elasticache-enable-at-rest-encryption",
	"AWS088": "aws-elasticache-enable-backup-retention",
	"AWS036": "aws-elasticache-enable-in-transit-encryption",
	"AWS005": "aws-elb-alb-not-public",
	"AWS083": "aws-elb-drop-invalid-headers",
	"AWS004": "aws-elb-http-not-used",
	"AWS010": "aws-elb-use-secure-tls-policy",
	"AWS037": "aws-iam-no-password-reuse",
	"AWS099": "aws-iam-no-policy-wildcards",
	"AWS042": "aws-iam-require-lowercase-in-passwords",
	"AWS041": "aws-iam-require-numbers-in-passwords",
	"AWS040": "aws-iam-require-symbols-in-passwords",
	"AWS043": "aws-iam-require-uppercase-in-passwords",
	"AWS039": "aws-iam-set-minimum-password-length",
	"AWS024": "aws-kinesis-enable-in-transit-encryption",
	"AWS019": "aws-kms-auto-rotate-keys",
	"AWS058": "aws-lambda-restrict-source-arn",
	"AWS022": "aws-msk-enable-in-transit-encryption",
	"AWS053": "aws-rds-enable-performance-insights",
	"AWS051": "aws-rds-encrypt-cluster-storage-data",
	"AWS052": "aws-rds-encrypt-instance-storage-data",
	"AWS003": "aws-rds-no-classic-resources",
	"AWS011": "aws-rds-no-public-db-access",
	"AWS091": "aws-rds-specify-backup-retention",
	"AWS094": "aws-redshift-encryption-customer-key",
	"AWS087": "aws-redshift-use-vpc",
	"AWS074": "aws-s3-block-public-acls",
	"AWS076": "aws-s3-block-public-policy",
	"AWS017": "aws-s3-enable-bucket-encryption",
	"AWS002": "aws-s3-enable-bucket-logging",
	"AWS077": "aws-s3-enable-versioning",
	"AWS073": "aws-s3-ignore-public-acls",
	"AWS001": "aws-s3-no-public-access-with-acl",
	"AWS075": "aws-s3-no-public-buckets",
	"AWS098": "aws-s3-specify-public-access-block",
	"AWS016": "aws-sns-enable-topic-encryption",
	"AWS015": "aws-sqs-enable-queue-encryption",
	"AWS047": "aws-sqs-no-wildcards-in-policy-documents",
	"AWS095": "aws-ssm-secret-use-customer-key",
	"AWS018": "aws-vpc-add-description-to-security-group",
	"AWS082": "aws-vpc-no-default-vpc",
	"AWS050": "aws-vpc-no-excessive-port-access",
	"AWS007": "aws-vpc-no-public-egress-sgr",
	"AWS009": "aws-vpc-no-public-egress-sgr",
	"AWS049": "aws-vpc-no-public-ingress-acl",
	"AWS006": "aws-vpc-no-public-ingress-sgr",
	"AWS008": "aws-vpc-no-public-ingress-sgr",
	"AWS084": "aws-workspaces-enable-disk-encryption",
	"AZU028": "azure-appservice-enforce-https",
	"AZU003": "azure-compute-enable-disk-encryption",
	"AZU006": "azure-container-configured-network-policy",
	"AZU008": "azure-container-limit-authorized-ips",
	"AZU009": "azure-container-logging",
	"AZU007": "azure-container-use-rbac-permissions",
	"AZU018": "azure-database-enable-audit",
	"AZU019": "azure-database-retention-period-set",
	"AZU025": "azure-datafactory-no-public-access",
	"AZU004": "azure-datalake-enable-at-rest-encryption",
	"AZU022": "azure-keyvault-content-type-for-secret",
	"AZU026": "azure-keyvault-ensure-key-expiry",
	"AZU023": "azure-keyvault-ensure-secret-expiry",
	"AZU021": "azure-keyvault-no-purge",
	"AZU020": "azure-keyvault-specify-network-acl",
	"AZU024": "azure-network-disable-rdp-from-internet",
	"AZU002": "azure-network-no-public-egress",
	"AZU001": "azure-network-no-public-ingress",
	"AZU017": "azure-network-ssh-blocked-from-internet",
	"AZU013": "azure-storage-allow-microsoft-service-bypass",
	"AZU012": "azure-storage-default-action-deny",
	"AZU014": "azure-storage-enforce-https",
	"AZU011": "azure-storage-no-public-access",
	"AZU016": "azure-storage-queue-services-logging-enabled",
	"AZU015": "azure-storage-use-secure-tls-policy",
	"AZU027": "azure-synapse-virtual-network-enabled",
	"DIG004": "digitalocean-compute-enforce-https",
	"DIG002": "digitalocean-compute-no-public-egress",
	"DIG001": "digitalocean-compute-no-public-ingress",
	"DIG003": "digitalocean-compute-use-ssh-keys",
	"DIG005": "digitalocean-spaces-acl-no-public-read",
	"DIG007": "digitalocean-spaces-disable-force-destroy",
	"DIG006": "digitalocean-spaces-versioning-enabled",
	"GEN002": "general-secrets-no-plaintext-exposure",
	"GEN001": "general-secrets-no-plaintext-exposure",
	"GEN005": "general-secrets-no-plaintext-exposure",
	"AWS044": "general-secrets-no-plaintext-exposure",
	"GEN003": "general-secrets-no-plaintext-exposure",
	"GIT001": "github-repositories-private",
	"GCP013": "google-compute-disk-encryption-no-plaintext-key",
	"GCP004": "google-compute-no-public-egress",
	"GCP003": "google-compute-no-public-ingress",
	"GCP009": "google-gke-enforce-pod-security-policy",
	"GCP007": "google-gke-metadata-endpoints-disabled",
	"GCP006": "google-gke-node-metadata-security",
	"GCP010": "google-gke-node-shielding-enabled",
	"GCP005": "google-gke-use-rbac-permissions",
	"GCP012": "google-gke-use-service-account",
	"GCP011": "google-iam-no-user-granted-permissions",
	"OCI001": "oracle-compute-no-public-ip",
}
