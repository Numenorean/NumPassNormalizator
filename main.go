package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
)

const (
	russianCode            = "7"
	russianTownCode        = "8"
	russianOperatorsCode   = "9"
	ukrainianCode          = "38"
	ukrainianOperatorsCode = "0"
	kazCode                = "7"

	addPlus = false

	notFoundStatus = "not_found"
)

var (
	findDigitsRegex = regexp.MustCompile("[0-9]+")
)

func main() {
	/*nums := []string{
		"79112563636",
		"380669867856",
		"0669863632",
		"9112563698",
		"89112563698",
		"7776984563",
		"+7 (911) 256-63-63",
		"38 066 987 45 66",
		"8 (987) 455 77 99",
		"kvs88@nur.kz",
	}
	for _, v := range nums {
		fmt.Println(normalizatePhone(v))
	}*/

	// Check if it has 2 args
	usage()
	filePath := os.Args[1]
	// Making result path. Adds _normalized before .txt
	resultPath := makeResultPath(filePath)
	// Removing existing file with normalized combos
	_ = os.Remove(resultPath)
	file := openFile(filePath)

	count, _ := lineCounter(file)
	fmt.Println("Lines count:", count)
	// Set up status bar
	bar := setUpBar(count)

	file = openFile(filePath)

	defer file.Close()

	scanner := bufio.NewScanner(file)

	resultArr := []string{}
	// Create count variable to save all of the lines
	// because we writes to file every 1000 lines
	// so in the end we miss around 1k lines
	n := 0
	// Read file line by line
	for scanner.Scan() {
		var phone, password string
		// Split by 2 delimiters (: and ;)
		combo := strings.FieldsFunc(scanner.Text(), func(r rune) bool {
			return r == ':' || r == ';'
		})
		// Check if combo does not has password
		// Check if password also has : or ;, so it would not be ignored
		if len(combo) < 2 {
			bar.Increment()
			n++
			continue
		} else if len(combo) > 2 {
			phone = combo[0]
			password = strings.Join(combo[1:], "")
		} else {
			phone, password = combo[0], combo[1]
		}
		normalizedPhone := normalizatePhone(phone)
		if normalizedPhone == notFoundStatus {
			bar.Increment()
			n++
			continue
		}
		// Creating slice and write to file normalizated combos every 1k lines
		resultArr = append(resultArr, normalizedPhone+":"+password)
		if len(resultArr) == 1000 || (count-n) < 1000 {
			writeHitsToFile(resultPath, strings.Join(resultArr, "\n"))
			// Clear slice
			resultArr = nil
		}
		bar.Increment()
		n++
	}
	bar.Finish()
}

func normalizatePhone(combo string) string {
	// Check if combo has digits, else return bad status
	if !hasDigits(combo) {
		return notFoundStatus
	}
	// Check if phone is correct, else return that as is without further checks
	comboLen := len(combo)
	if (comboLen == 12 || comboLen == 11) && hasPrefixes(combo, russianCode, ukrainianCode) {
		return combo
	}
	// If combos has smth else except digits, extract them via regex
	if !isDigits(combo) {
		combo = extractDigits(combo)
		comboLen = len(combo)
	}
	// Check if it is a ukrainian phone
	if comboLen == 12 && strings.HasPrefix(combo, ukrainianCode) {
		return combo
	}

	// 11 it's default length of phones like this 89112563636
	// 10 it's default lenght of phones like this 9112563636
	// hasPrefixes checks if one of the variants will be found in the start of phone
	if (comboLen == 11 || comboLen == 10) && hasPrefixes(combo, russianCode,
		russianTownCode, russianOperatorsCode, ukrainianOperatorsCode) {
		switch combo[0] {
		// Check if phone starts from 8 code
		case russianTownCode[0]:
			combo = russianCode + combo[1:]
		// Check if phone starts from 0 code (ukrainian default operators code)
		case ukrainianOperatorsCode[0]:
			combo = ukrainianCode + combo
		// Kaz phone check
		case russianCode[0]:
			if comboLen == 10 {
				combo = russianCode + combo
			}
		// Check if phone starts from operators code (for russia it is 9)
		case russianOperatorsCode[0]:
			combo = russianCode + combo
		}
		// Otherwise determines that as any russian operator code (just add 7)
	} else if comboLen == 10 {
		combo = russianCode + combo
	} else {
		return notFoundStatus
	}

	if addPlus {
		return "+" + combo
	}
	return combo

}

func hasPrefixes(s string, prefixes ...string) bool {
	for _, sub := range prefixes {
		if strings.HasPrefix(s, sub) {
			return true
		}
	}
	return false
}

func isDigits(s string) bool {
	for _, v := range s {
		if v < '0' || v > '9' {
			return false
		}
	}
	return true
}

func hasDigits(s string) bool {
	for _, v := range s {
		if v > '0' || v <= '9' {
			return true
		}
	}
	return false
}

func extractDigits(s string) string {
	digitsSlice := findDigitsRegex.FindAllString(s, -1)
	if len(digitsSlice) == 0 {
		return notFoundStatus
	}
	return strings.Join(digitsSlice, "")
}

func openFile(path string) *os.File {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func setUpBar(count int) *pb.ProgressBar {
	bar := pb.Full.Start(count)
	bar.SetRefreshRate(time.Millisecond)

	tmpl := `{{string . "prefix"}}{{counters . }} {{bar . }} {{percent . }} {{speed . }} {{etime . "%s"}}/{{rtime . "%s"}}`

	bar.SetTemplateString(tmpl)
	return bar
}

func makeResultPath(path string) string {
	arr := []string{}
	splitted := strings.Split(path, ".")
	arr = append(arr,
		strings.Join(splitted[0:len(splitted)-1], ""),
		"_normalizated",
		"."+splitted[len(splitted)-1])
	return strings.Join(arr, "")
}

func usage() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s path/to/file\n", os.Args[0])
		fmt.Scanln()
		os.Exit(1)
	}
}

func writeHitsToFile(resultPath, data string) {
	f, err := os.OpenFile(resultPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	f.Write([]byte(data + "\n"))
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
