package main

import (
	"os"
	"strconv"
)

func removeFromSlice(slice []string, v string) []string {
	for i, s := range slice {
		if s == v {
			return append(slice[:i], slice[i+1])
		}
	}
	return slice
}

func getEnv(name, defaultValue string) (int, error) {
	var err error
	var envVar int
	envVarString := os.Getenv(name)
	if envVarString == "" {
		envVar, err = strconv.Atoi(defaultValue)
		if err != nil {
			return 0, err
		}
		return envVar, nil
	}

	envVar, err = strconv.Atoi(envVarString)
	if err != nil {
		return 0, err
	}

	return envVar, nil
}
