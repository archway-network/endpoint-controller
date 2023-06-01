package utils

import (
	"os"
	"strconv"
)

func RemoveFromSlice(s []string, v string) []string {
	for i, x := range s {
		if x == v {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
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
