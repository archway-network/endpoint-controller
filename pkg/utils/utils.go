package utils

import (
	"os"
	"strconv"
)

func RemoveFromSlice(slice []string, v string) []string {
	for i, s := range slice {
		if s == v {
			return append(slice[:i], slice[i+1])
		}
	}
	return slice
}

func GetEnv(name, defaultValue string) (int, error) {
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
