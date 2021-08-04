package util

const (
	prefix                         = "harvesterhci.io"
	RemovedPvcsAnnotationKey       = prefix + "/removedPersistentVolumeClaims"
	AnnotationMigrationTarget      = prefix + "/migrationTargetNodeName"
	AnnotationMigrationUID         = prefix + "/migrationUID"
	AnnotationMigrationState       = prefix + "/migrationState"
	AnnotationTimestamp            = prefix + "/timestamp"
	AnnotationVolumeClaimTemplates = prefix + "/volumeClaimTemplates"
)
