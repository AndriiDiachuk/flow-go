package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

func AssertNoError(err error, panicMessage string) {
	if err != nil {
		panic(panicMessage + ": " + err.Error())
	}
}

// if doesn't have trailing slash, append one
// if has trailing slash, doesn't change it
func AddTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}

func ConvertToNDecimalPlaces2(n int, numerator float32, denominator int) float32 {
	return convertToNDecimalPlacesInternal(n, numerator, float32(denominator))
}

func ConvertToNDecimalPlaces(n int, numerator, denominator int) float32 {
	return convertToNDecimalPlacesInternal(n, float32(numerator), float32(denominator))
}

func convertToNDecimalPlacesInternal(n int, numerator, denominator float32) float32 {
	formatSpecifier := "%." + fmt.Sprint(n) + "f"
	ratioString := fmt.Sprintf(formatSpecifier, numerator/denominator)
	ratioFloat, err := strconv.ParseFloat(ratioString, 32)
	AssertNoError(err, "failure parsing string to float")
	return float32(ratioFloat)
}

// from https://stackoverflow.com/a/30708914/5719544
func IsDirEmpty(name string) bool {
	f, err := os.Open(name)
	AssertNoError(err, "error reading directory")

	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	AssertNoError(err, "error reading dir contents")
	return false
}

func DirExists(path string) bool {
	_, err := os.Stat(path)

	// directory exists if there is no error
	if err == nil {
		return true
	}

	// directory doesn't exist if error is of specific type
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}

	// should never get to here
	panic("error checking if directory exists:" + err.Error())
}

// save test run/summary to local JSON file
func SaveToFile(fileName string, testSummary interface{}) {
	testSummaryBytes, err := json.MarshalIndent(testSummary, "", "  ")
	AssertNoError(err, "error marshalling json")

	file, err := os.Create(fileName)
	AssertNoError(err, "error creating filename")
	defer file.Close()

	_, err = file.Write(testSummaryBytes)
	AssertNoError(err, "error saving test summary to file")
}
