<div align="center">
<h1>tickler</h1>

[Tickler](https://github.com/goodjobtech/tickler), is a library for long-running services to **enqueue** and **process** the jobs in the background with **simplicity** and **performance** in mind. 


</div>

## Installation

With Go module support (Go 1.11+), simply add the following import
```go
import "github.com/goodjobtech/tickler"
```

Otherwise, to install the tickler, run the following command:

```shell
$ go get -u github.com/goodjobtech/tickler
```

## Usage

### Enqueue a job

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	tl := tickler.New()

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}

	tl.Enqueue(request)
}
```

### Enqueue a job with context

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	scheduler := tickler.New()

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}
    
	ctx = context.WithValue(ctx, "name", "tickler")
	tl.EnqueueWithContext(ctx, request)
}
```

### Wait for another job to be completed

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
	"time"
)

func main() {
	tl := tickler.New()

	request := tickler.Request{
		Job: func() error {
			time.Sleep(time.Second * 5)
			return nil
		},
		Name: "cpu-heavy-job",
	}

	tl.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("CPU heavy job is done!")
			return nil
		},
		Name: "waiting",
	}

	tl.Enqueue(waitRequest, tickler.WaitFor("cpu-heavy-job"))
}
```

### Enqueue a job if a given job is successfully completed

If a job returns with no error, i.e. `err == nil`, it is succeeded.

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	tl := tickler.New()

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}

	tl.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("I am waiting")
			return nil
		},
		Name: "waiting",
	}
	
	// This will be executed
	tl.Enqueue(waitRequest, tickler.IfSuccess("hello-world"))
}
```

### Enqueue a job if a given job is failed

If a job returns with error, i.e. `err != nil`, it is failed.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	tl := tickler.New()

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return errors.New("I failed")
		},
		Name: "hello-world",
	}

	tl.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("I am waiting")
			return nil
		},
		Name: "waiting",
	}

	// This will be executed
	tl.Enqueue(waitRequest, tickler.IfFailure("hello-world"))
}
```

### Retry a failed job

*tickler* uses exponential backoff mechanism to retry failed jobs

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	tl := tickler.New()

	counter := 3

	request := tickler.Request{
		Job: func() error {
			if counter == 0 {
				fmt.Println("beat that counter")
				return nil
			}

			counter--
			return errors.New("still got some counter")
		},
		Name: "retry",
	}

	// This job will be called maximum 4 times.
	tl.Enqueue(request, tickler.WithRetry(4))
}
```

## Contributing

All kinds of pull request and feature requests are welcomed!

## License

Tickler's source code is licensed under [MIT License](https://choosealicense.com/licenses/mit/).
