package db

import "github.com/google/uuid"

// ProjectNamespace is the fixed UUID v5 namespace for fileditto file identity.
// This value MUST NEVER change — changing it would alter the UUID of every
// existing file and break all foreign key relationships.
const ProjectNamespace = "a8b4c6d2-e1f9-4a5b-88c7-d3e9f2a1b4c6"

var projectNamespaceUUID = uuid.MustParse(ProjectNamespace)

// FileUUID derives a deterministic UUID v5 for a file from its SHA-256 hex digest.
// The same sha256hex input always produces the same UUID.
func FileUUID(sha256hex string) (uuid.UUID, error) {
	return uuid.NewSHA1(projectNamespaceUUID, []byte(sha256hex)), nil
}

// NameUUID derives a deterministic UUID v5 for any named entity within the fileditto namespace.
func NameUUID(name string) uuid.UUID {
	return uuid.NewSHA1(projectNamespaceUUID, []byte(name))
}
