import uuid

# Project namespace UUID for UUID v5 file identity generation.
# This value MUST NEVER change — it is the same constant used in the Go code.
# Changing it would alter the UUID of every file and break all foreign key relationships.
NAMESPACE_UUID = uuid.UUID("a8b4c6d2-e1f9-4a5b-88c7-d3e9f2a1b4c6")


def file_uuid(sha256_hex: str) -> str:
    """Return a deterministic UUID v5 string for a file given its SHA-256 hex digest.

    The same sha256_hex input always produces the same UUID, and the result is
    identical to the UUID produced by the Go FileUUID function for the same input.
    """
    return str(uuid.uuid5(NAMESPACE_UUID, sha256_hex))
