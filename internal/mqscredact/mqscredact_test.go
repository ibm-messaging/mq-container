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
	"strings"
	"testing"
)

const passwordString = passwordHalf1 + passwordHalf2
const passwordHalf1 = "hippo"
const passwordHalf2 = "123456"

var testStrings = [...]string{
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('" + passwordString + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordString + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD     ('" + passwordString + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD\t\t('" + passwordString + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) ldappwd('" + passwordString + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LdApPwD('" + passwordString + "')",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD('" + passwordString + "')",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD(\"" + passwordString + "\")",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD     ('" + passwordString + "')",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD\t\t('" + passwordString + "')",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) password('" + passwordString + "')",
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) pAsSwOrD('" + passwordString + "')",
	"ALTER QMGR SSLCRYP('" + passwordString + "')",
	"ALTER QMGR SSLCRYP(\"" + passwordString + "\")",
	"ALTER QMGR SSLCRYP     ('" + passwordString + "')",
	"ALTER QMGR SSLCRYP\t\t('" + passwordString + "')",
	"ALTER QMGR sslcryp('" + passwordString + "')",
	"ALTER QMGR sslCRYP('" + passwordString + "')",

	// Line continuation ones
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "+\n " + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "+\n\t" + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "+\n\t   " + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('" + passwordHalf1 + "+\n " + passwordHalf2 + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('" + passwordHalf1 + "+\n\t" + passwordHalf2 + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('" + passwordHalf1 + "+\n\t   " + passwordHalf2 + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "-\n" + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD('" + passwordHalf1 + "-\n" + passwordHalf2 + "')",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "+  \n " + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "+\t\n " + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "-  \n" + passwordHalf2 + "\")",
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD(\"" + passwordHalf1 + "-\t\n" + passwordHalf2 + "\")",

	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD(\"" + passwordHalf1 + "+\n " + passwordHalf2 + "\")",

	"ALTER QMGR SSLCRYP(\"" + passwordHalf1 + "+\n " + passwordHalf2 + "\")",

	//edge cases
	"ALTER QMGR SSLCRYP(\"" + passwordHalf1 + "+\n 123+\n 456\")",
	"ALTER QMGR SSLCRYP(\"" + passwordHalf1 + "-\n123-\n456\")",

	"ALTER QMGR SSLCRYP(\"" + passwordHalf1 + "+\n 1+\n 2+\n 3+\n   4+\n  5+\n  6\")",
	"ALTER QMGR SSLCRYP(\"" + passwordHalf1 + "-\n1-\n2-\n3-\n4-\n5-\n6\")",

	"ALTER QMGR SSLCRYP  + \n  (\"" + passwordHalf1 + "+\n 1+\n 2+\n 3+\n   4+\n  5+\n  6\")",
	"ALTER QMGR SSLCRYP  -  \n(\"" + passwordHalf1 + "-\n1-\n2-\n3-\n4-\n5-\n6\")",

	"ALTER QMGR SSL  +  \n    CRYP(\"" + passwordHalf1 + "+\n 1+\n 2+\n 3+\n   4+\n  5+\n  6\")",
	"ALTER QMGR SSL  -     \nCRYP(\"" + passwordHalf1 + "-\n1-\n2-\n3-\n4-\n5-\n6\")",

	"ALTER QMGR +   \n   SSL +\n CRYP(\"" + passwordHalf1 + "+\n 1+\n 2+\n 3+\n   4+\n  5+\n  6\") +\n TEST(1234)",
	"ALTER QMGR    -\nSSL -\nCRYP(\"" + passwordHalf1 + "-\n1-\n2-\n3-\n4-\n5-\n6\") -\nTEST(1234)",

	"ALTER QMGR +\n * COMMENT\n SSL +\n * COMMENT IN MIDDLE\n CRYP('" + passwordString + "')",

	" 1: ALTER CHANNEL(TEST2) CHLTYPE(SDR) PASS+\n   : *test comment\n   : WORD('" + passwordString + "')",
	" 2: ALTER CHANNEL(TEST3) CHLTYPE(SDR) PASSWORD('" + passwordHalf1 + "-\n*commentinmiddle with ' \n" + passwordHalf2 + "')",
	" 3: ALTER CHANNEL(TEST3) CHLTYPE(SDR) PASSWORD('" + passwordHalf1 + "-\n*commentinmiddle with ') \n" + passwordHalf2 + "')",
}

var expected = [...]string{
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD     " + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD\t\t" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) ldappwd" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LdApPwD" + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD" + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD" + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD     " + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD\t\t" + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) password" + RedactionString,
	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) pAsSwOrD" + RedactionString,
	"ALTER QMGR SSLCRYP" + RedactionString,
	"ALTER QMGR SSLCRYP" + RedactionString,
	"ALTER QMGR SSLCRYP     " + RedactionString,
	"ALTER QMGR SSLCRYP\t\t" + RedactionString,
	"ALTER QMGR sslcryp" + RedactionString,
	"ALTER QMGR sslCRYP" + RedactionString,

	// Line continuation ones
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,
	"DEFINE AUTHINFO(TEST) AUTHTYPE(IDPWLDAP) LDAPPWD" + RedactionString,

	"DEFINE CHANNEL(CHL) CHLTYPE(SOMETHING) PASSWORD" + RedactionString,

	"ALTER QMGR SSLCRYP" + RedactionString,

	//edge cases
	"ALTER QMGR SSLCRYP" + RedactionString,
	"ALTER QMGR SSLCRYP" + RedactionString,

	"ALTER QMGR SSLCRYP" + RedactionString,
	"ALTER QMGR SSLCRYP" + RedactionString,

	"ALTER QMGR SSLCRYP  + \n    \t  " + RedactionString,
	"ALTER QMGR SSLCRYP  -  \n    " + RedactionString,

	"ALTER QMGR SSL  +  \n    CRYP" + RedactionString,
	"ALTER QMGR SSL  -     \nCRYP" + RedactionString,

	"ALTER QMGR +   \n   SSL +\n CRYP" + RedactionString + " +\n TEST(1234)",
	"ALTER QMGR    -\nSSL -\nCRYP" + RedactionString + " -\nTEST(1234)",

	"ALTER QMGR +\n * COMMENT\n SSL +\n * COMMENT IN MIDDLE\n CRYP" + RedactionString,

	"1: ALTER CHANNEL(TEST2) CHLTYPE(SDR) PASS+\n   : *test comment\n   : WORD" + RedactionString,
	"2: ALTER CHANNEL(TEST3) CHLTYPE(SDR) PASSWORD" + RedactionString,
	"3: ALTER CHANNEL(TEST3) CHLTYPE(SDR) PASSWORD" + RedactionString,
}

// Returns true if the 2 strings are equal ignoring whitespace characters
func compareIgnoreWhiteSpace(str1, str2 string) bool {
	whiteSpaces := [...]string{" ", "\t", "\n", "\r"}
	for _, w := range whiteSpaces {
		str1 = strings.Replace(str1, w, "", -1)
		str2 = strings.Replace(str2, w, "", -1)
	}

	return str1 == str2
}

func TestAll(t *testing.T) {
	for i, v := range testStrings {
		back, _ := Redact(v)
		if strings.Contains(back, passwordHalf1) || strings.Contains(back, passwordHalf2) || strings.Contains(back, passwordString) {
			t.Errorf("MAJOR FAIL[%d]: Found an instance of the password. ", i)
		}

		if !compareIgnoreWhiteSpace(back, expected[i]) {
			t.Errorf("FAIL[%d]:\nGave    :%s\nexpected:%s\ngot     :%s", i, v, expected[i], back)
		}
	}
}
