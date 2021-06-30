package main

import "os"

func AtomMetadata() map[string]string {
	return map[string]string{
		"ATOM_CLIENT_SDK_VERSION": "2.0.0-RELEASE",
		"ATOM_GROUP_ID":           os.Getenv("atom_group_id"),
		"ATOM_NAMESPACE_ID":       os.Getenv("atom_namespace_id"),
		"ATOM_CLUSTER_ID":         os.Getenv("atom_cluster_id"),
		"ATOM_INSTANCE_ID":        os.Getenv("atom_instance_id"),
	}
}
