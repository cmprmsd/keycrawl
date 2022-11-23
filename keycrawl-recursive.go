package main

// Author Sebastian @cmprmsd Haas
import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

var mails []string

//var charliststr = "za " // for debugging
var charliststr = "abcdefghijklmnopqrstuvwxyz.-_ "
var charlistrune = []rune(charliststr)
var totalRequests = 0

func requestNames(url string, char string) []string {
	// text extractors
	color.New(color.FgBlue).Fprintf(os.Stderr, "requesting %v\n", char)
	email := regexp.MustCompile("pgpUserID: (.+)<(.+)>") // <- the more generic way
	// build arglist
	args := []string{"-E", "pr=5/noprompt", "-H", url, "-b", os.Args[2], "-z", "0", "-x", "(pgpUserID=" + char + "*)", "-LL", "-o", "ldif-wrap=no"}
	// disable TLS validation
	os.Setenv("LDAPTLS_REQCERT", "never")
	// execute ldapsearch and read stdout and stderr
	cmd := exec.Command("ldapsearch", args...)
	totalRequests++
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	// check if response size limit was reached (too many results for one request)
	var mailsForChar []string
	if stderr.String() != "" {
		return []string{"many"}
	} else {
		match := email.FindAllStringSubmatch(out.String(), -1)
		if match == nil {
			log.Printf("No emails with the beginning: %v ", char)
			return []string{""}
		}
		for _, s := range match {
			// clean up names [1] and emails [2]
			s[1] = strings.TrimSpace(s[1])
			s[2] = strings.TrimSpace(s[2])
			// output each match found:
			fmt.Printf("{\"name\":\"%s\", \"email\":\"%s\"}\n", s[1], s[2])
			mailsForChar = append(mailsForChar, s[2])
		}
	}
	fmt.Fprintf(os.Stderr, "Return value: %v\n", mailsForChar)
	return mailsForChar
}

func letters(chars string) []string {
	url := os.Args[1]
	result := requestNames(url, string(chars))
	if result[0] == "many" {
		// limit exceeded. Need to go deeper
		for n := 0; n < len(charlistrune); n++ {
			letters(string(chars) + string(charlistrune[n]))
		}
	} else if result[0] != "" {
		mails = append(mails, result...)
	}
	return mails
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// sig is a ^C, handle it
			log.Printf("Got: %v", sig)
			log.Printf("Stats:\n\nTotal requests: %v\nIdentified e-mails: %v\n", totalRequests, len(mails))
			os.Exit(0)
		}
	}()
	for first := 0; first < len(charlistrune); first++ {
		mails = letters(string(charlistrune[first]))
	}
	log.Printf("Stats:\n\nTotal requests: %v\nIdentified e-mails: %v\n", totalRequests, len(mails))

}
