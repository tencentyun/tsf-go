package hash

import (
	"fmt"
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	c := NewHash()
	c.Add(Node{"cacheA", 0})
	c.Add(Node{"cacheB", 1})
	c.Add(Node{"cacheC", 2})
	users := []string{"user_mcnulty", "user_bunk", "user_omar", "user_bunny", "user_stringer"}
	for _, u := range users {
		server, err := c.Get(u)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s => %s %d\n", u, server, c.Index(server))
	}
	fmt.Println("add node z!")
	c.Add(Node{"cacheZ", 3})
	c.Add(Node{"cacheD", 4})

	for _, u := range users {
		server, err := c.Get(u)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s => %s %d\n", u, server, c.Index(server))
	}
}
