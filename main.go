package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/go-ldap/ldap/v3"
	"github.com/icrowley/fake"
)

type Person struct {
	Name    string
	Surname string
}

var bold = lipgloss.NewStyle().Bold(true)
var boldRed = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff0000"))

func (p Person) FullName() string {
	return fmt.Sprintf("%s %s", p.Name, p.Surname)
}

func (p Person) Email() string {
	return fmt.Sprintf("%s.%s@example.org", strings.ToLower(p.Name), strings.ToLower(p.Surname))
}

func FakePerson() Person {
	return Person{
		Name:    fake.FirstName(),
		Surname: fake.LastName(),
	}
}

func GenerateFakeData(amount int) []Person {
	var people = make([]Person, amount)
	for i := 0; i < amount; i++ {
		people[i] = FakePerson()
	}
	return people
}

func main() {
	fmt.Println(bold.Render("Connecting to LDAP server..."))
	conn, err := net.Dial("tcp", "localhost:389")
	if err != nil {
		panic(err)
	}

	ldapConnection := ldap.NewConn(conn, false)
	ldapConnection.Start()
	defer ldapConnection.Close()

	// Bind
	err = ldapConnection.Bind("cn=admin,dc=example,dc=org", "admin")
	handleErr(err)

	fmt.Println(bold.Render("\nGenerating fake data..."))
	people := GenerateFakeData(10)

	fmt.Println(bold.Render("\nAdding fake data..."))
	err = addMany(ldapConnection, people)
	handleErr(err)

	fmt.Println(bold.Render("\nReading all data..."))
	searchResult, err := readAll(ldapConnection)
	handleErr(err)

	// Print the result
	printResult(searchResult, true)

	// Pick a random person
	randomPerson := people[0]
	// Remove the person from the list
	people = people[1:]

	fmt.Println(bold.Render("\nSearching for person " + randomPerson.FullName()))
	searchResult, err = readSingle(ldapConnection, randomPerson)
	handleErr(err)

	// Print the person
	printResult(searchResult, false)

	fmt.Println(bold.Render("\nRemoving person " + randomPerson.FullName()))
	err = remove(ldapConnection, randomPerson)
	handleErr(err)
	fmt.Printf("Person %s was removed\n", randomPerson.FullName())

	fmt.Println(bold.Render("\nSearching for person " + randomPerson.FullName()))
	searchResult, err = readSingle(ldapConnection, randomPerson)
	handleErr(err)
	if len(searchResult.Entries) > 0 {
		fmt.Println(boldRed.Render("Person was not removed"))
		return
	} else {
		fmt.Printf("Person %s was successfully removed\n", randomPerson.FullName())
	}

	newPerson := Person{
		Name:    "John",
		Surname: "Xina",
	}

	fmt.Println(bold.Render("\nSearching for new, not existing person " + newPerson.FullName()))
	searchResult, err = readSingle(ldapConnection, newPerson)
	handleErr(err)
	if len(searchResult.Entries) > 0 {
		fmt.Println(boldRed.Render("Person already exists"))
		return
	} else {
		fmt.Printf("Person %s was not found\n", newPerson.FullName())
	}

	printResult(searchResult, false)

	fmt.Println(bold.Render("\nAdding new person " + newPerson.FullName()))
	err = add(ldapConnection, newPerson)
	handleErr(err)
	fmt.Printf("Person %s was added\n", newPerson.FullName())

	fmt.Println(bold.Render("\nSearching for new person " + newPerson.FullName()))
	searchResult, err = readSingle(ldapConnection, newPerson)
	handleErr(err)
	if len(searchResult.Entries) > 0 {
		fmt.Printf("Person %s was found\n", newPerson.FullName())
	} else {
		fmt.Println(boldRed.Render("Person was not added"))
		return
	}

	// Print the result
	printResult(searchResult, false)

	fmt.Println()

	// Wipe all data
	err = removeMany(ldapConnection, people)
	handleErr(err)
	err = remove(ldapConnection, newPerson)
	handleErr(err)
	fmt.Println(bold.Render("\nAll data was removed"))
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(boldRed.Render("Error: " + err.Error()))
		os.Exit(1)
	}
}

func printResult(result *ldap.SearchResult, onlyDn bool) {
	fmt.Printf("Found %d item(s):\n", len(result.Entries))
	for _, entry := range result.Entries {
		fmt.Printf("dn: %s\n", entry.DN)
		if onlyDn {
			continue
		}
		for _, attr := range entry.Attributes {
			fmt.Printf("\t%s", attr.Name)
			for _, value := range attr.Values[:len(attr.Values)-1] {
				fmt.Printf(": %s", value)
			}
			fmt.Printf(": %s", attr.Values[len(attr.Values)-1])
			println()
		}
	}
}

func add(conn *ldap.Conn, person Person) error {
	addRequest := ldap.NewAddRequest(fmt.Sprintf("cn=%s,dc=example,dc=org", person.FullName()), nil)
	addRequest.Attribute("objectClass", []string{"inetOrgPerson"})
	addRequest.Attribute("cn", []string{person.FullName()})
	addRequest.Attribute("givenName", []string{person.Name})
	addRequest.Attribute("sn", []string{person.Surname})
	addRequest.Attribute("mail", []string{person.Email()})
	err := conn.Add(addRequest)
	if err != nil {
		if ldapError, ok := err.(*ldap.Error); ok {
			if ldapError.ResultCode != ldap.LDAPResultEntryAlreadyExists {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func addMany(conn *ldap.Conn, people []Person) error {
	for _, person := range people {
		if err := add(conn, person); err != nil {
			return err
		}
	}
	return nil
}

func remove(conn *ldap.Conn, person Person) error {
	delRequest := ldap.NewDelRequest(fmt.Sprintf("cn=%s,dc=example,dc=org", person.FullName()), nil)
	return conn.Del(delRequest)
}

func removeMany(conn *ldap.Conn, people []Person) error {
	for _, person := range people {
		if err := remove(conn, person); err != nil {
			return err
		}
	}
	return nil
}

func readAll(conn *ldap.Conn) (*ldap.SearchResult, error) {
	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=org",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"*"},
		nil,
	)
	return conn.Search(searchRequest)
}

func readSingle(conn *ldap.Conn, person Person) (*ldap.SearchResult, error) {
	searchRequest := ldap.NewSearchRequest(
		"dc=example,dc=org",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", person.FullName()),
		[]string{"*"},
		nil,
	)
	return conn.Search(searchRequest)
}
