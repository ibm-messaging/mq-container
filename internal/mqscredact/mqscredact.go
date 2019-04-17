/*
© Copyright IBM Corporation 2019

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mqscredact

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

/* List of sensitive MQ Parameters */
var sensitiveParameters = []string{"LDAPPWD", "PASSWORD", "SSLCRYP"}

// redactionString is what sensitive paramters will be replaced with
const redactionString = "(*********)"

func findEndOfParamterString(stringDenoter rune, r *bufio.Reader) string {
	parameter := ""
	for {
		char, _, err := r.ReadRune()
		if err != nil {
			return parameter
		}
		parameter = parameter + string(char)
		if char == stringDenoter {
			break
		} else if char == '\n' {
			// Check if we're on a comment line
		NewLineLoop:
			for {
				// Look at next character without moving buffer forwards
				chars, err := r.Peek(1)
				if err != nil {
					return parameter
				}
				// Check if we're at the beginning of some data.
				startOutput, _ := regexp.MatchString(`[^:0-9\s]`, string(chars[0]))
				if startOutput {
					// We are at the start, check if we're on a comment line
					if chars[0] == '*' {
						// found a comment line. go to the next newline chraracter
					CommentLoop:
						for {
							char, _, err = r.ReadRune()
							if err != nil {
								return parameter
							}
							parameter = parameter + string(char)
							if char == '\n' {
								break CommentLoop
							}
						}
						// Go round again as we're now on a new line
						continue NewLineLoop
					}
					// We've checked for comment and it isn't a comment line so break without moving buffer forwards
					break NewLineLoop
				}
				// Move the buffer forward and try again
				char, _, _ = r.ReadRune()
				parameter = parameter + string(char)
			}
		}
	}

	return parameter
}

// getParameterString reads from r in order to find the end of the MQSC Parameter value. This is enclosed in ( ).
// This function will return what it finds and will increment the reader pointer along as it goes.
func getParameterString(r *bufio.Reader) string {
	// Add the ( in as it will have been dropped before.
	parameter := "("
Loop:
	for {
		char, _, err := r.ReadRune()
		if err != nil {
			return parameter
		}

		parameter = parameter + string(char)

		switch char {
		case ')':
			break Loop
		// TODO: Duplicate code..
		case '\'', '"':
			parameter = parameter + findEndOfParamterString(char, r)
		}
	}

	return parameter
}

func resetAllParameters(currentVerb, originalString *string, lineContinuation, foundGap, parameterNext, redacting, checkComment *bool) {
	*currentVerb = ""
	*originalString = ""
	*lineContinuation = false
	*foundGap = false
	*parameterNext = false
	*redacting = false
	*checkComment = true
}

// Redact is the main function for redacting sensitive parameters in MQSC strings
// It accepts a string and redacts sensitive paramters such as LDAPPWD or PASSWORD
func Redact(out string) (string, error) {
	out = strings.TrimSpace(out)
	var returnStr, currentVerb, originalString string
	var lineContinuation, foundGap, parameterNext, redacting, checkComment bool
	newline := true
	resetAllParameters(&currentVerb, &originalString, &lineContinuation, &foundGap, &parameterNext, &redacting, &checkComment)
	r := bufio.NewReader(strings.NewReader(out))

MainLoop:
	for {
		// We have found a opening ( so use special parameter parsing
		if parameterNext {
			parameterStr := getParameterString(r)
			if !redacting {
				returnStr = returnStr + parameterStr
			} else {
				returnStr = returnStr + redactionString
			}

			resetAllParameters(&currentVerb, &originalString, &lineContinuation, &foundGap, &parameterNext, &redacting, &checkComment)
		}

		// Loop round getting hte next parameter
		char, _, err := r.ReadRune()
		if err == io.EOF {
			if originalString != "" {
				returnStr = returnStr + originalString
			}
			break
		} else if err != nil {
			return returnStr, err
		}

		/* We need to push forward until we find a non-whitespace, digit or colon character */
		if newline {
			startOutput, _ := regexp.MatchString(`[^:0-9\s]`, string(char))
			if !startOutput {
				originalString = originalString + string(char)
				continue MainLoop
			}
			newline = false
		}

		switch char {
		// Found a line continuation character
		case '+', '-':
			lineContinuation = true
			foundGap = false
			originalString = originalString + string(char)
			continue MainLoop

		// Found whitespace/new line
		case '\n':
			checkComment = true
			newline = true
			fallthrough
		case '\t', '\r', ' ':
			if !lineContinuation {
				foundGap = true
			}
			originalString = originalString + string(char)
			continue MainLoop

		// Found a paramter value
		case '(':
			parameterNext = true
			/* Do not continue as we need to do some checks */

		// Found a comment, parse in a special manner
		case '*':
			if checkComment {
				originalString = originalString + string(char)
				// Loop round until we find the new line character that marks the end of the comment
			CommentLoop:
				for {
					char, _, err := r.ReadRune()
					if err == io.EOF {
						if originalString != "" {
							returnStr = returnStr + originalString
						}
						break MainLoop
					} else if err != nil {
						return returnStr, err
					}
					originalString = originalString + string(char)

					if char == '\n' {
						break CommentLoop
					}

				}

				//Comment has been read and added to original string, go back to start
				checkComment = true
				newline = true
				continue MainLoop
			}
			/* Do not continue as we need to do some checks */

		} //end of switch

		checkComment = false

		if lineContinuation {
			lineContinuation = false
		}
		if foundGap || parameterNext {
			// we've completed an parameter so check whether it is sensitive
			currentVerb = strings.ToUpper(currentVerb)

			if isSensitiveCommand(currentVerb) {
				redacting = true
			}

			// Add the unedited string to the return string
			returnStr = returnStr + originalString

			//reset some of the parameters
			originalString = ""
			currentVerb = ""
			foundGap = false
			lineContinuation = false
		}

		originalString = originalString + string(char)
		currentVerb = currentVerb + string(char)
	}

	return returnStr, nil
}

// isSensitiveCommand checks whether the given string contains a sensitive parameter.
// We use contains here because we can't determine whether a line continuation seperates
// parts of a parameter or two different parameters.
func isSensitiveCommand(command string) bool {
	for _, v := range sensitiveParameters {
		if strings.Contains(command, v) {
			return true
		}
	}

	return false
}
