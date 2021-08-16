## Install

```shell
go get github.com/wksw/gopool
```

## Use

```golang
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/wksw/gopool"
)

type task struct {
	name string
}

func (t *task) String() string {
	return t.name
}

func handler(task interface{}) {
	time.Sleep(1 * time.Second)
	log.Println("handle task", task)
}

func status(pool *gopool.Pool) {
	fmt.Println("workers", pool.Workers())
	fmt.Println("capacity", pool.Capacity())
	fmt.Println("runningJobs", pool.RunningJobs())
}

func main() {
	pool, err := gopool.New(10, 3, handler)
	if err != nil {
		log.Fatal(err.Error())
	}
	pool.WithExitCallback(func() {
		fmt.Println("pool closed")
	})
	for i := 0; i < 10; i++ {
		pool.Add(&task{name: fmt.Sprintf("task_%d", i)})
	}
	fmt.Println("close pool")
	pool.Close()
	for i := 0; i < 10; i++ {
		if err := pool.Add(&task{name: fmt.Sprintf("task_close_%d", i)}); err != nil {
			fmt.Println("add task fail", err.Error())
		}
	}
}
```
