<div align="center">
<h1>tickler</h1>

[tickler](https://github.com/goodjobtech/tickler), Background job scheduler library for Golang

</div>

## Installation

You can import tickler to your project as:

```shell
go get github.com/goodjobtech/tickler
go install github.com/goodjobtech/tickler
```

## Usage

### Create a new Tickler instance

```go
package main

import (
	"context"
	"github.com/goodjobtech/tickler"
)

func main() {
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit
	
	scheduler := tickler.New(ctx, limit)
	
	// Starts service loop to process jobs
	scheduler.Start()
	
	// Stops service loop to process next jobs
	scheduler.Stop()
}
```

### Enqueue a job

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}

	scheduler.Enqueue(request)
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
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}
    
	ctx = context.WithValue(ctx, "name", "tickler")
	scheduler.EnqueueWithContext(ctx, request)
}
```

### Wait for a job to be completed

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}

	scheduler.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("I am waiting")
			return nil
		},
		Name: "waiting",
	}

	scheduler.Enqueue(waitRequest, tickler.WaitForJobs("hello-world"))
}
```

### Process a job if its parent job is successfully completed

If a job returns with no error, i.e. `err == nil`, it is succeeded.

```go
package main

import (
	"context"
	"fmt"
	"github.com/goodjobtech/tickler"
)

func main() {
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return nil
		},
		Name: "hello-world",
	}

	scheduler.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("I am waiting")
			return nil
		},
		Name: "waiting",
	}
	
	// This will be executed
	scheduler.Enqueue(waitRequest, tickler.IfSuccess("hello-world"))
}
```

### Process a job if its parent job is failed

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
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

	request := tickler.Request{
		Job: func() error {
			fmt.Println("Hello World!")
			return errors.New("I failed")
		},
		Name: "hello-world",
	}

	scheduler.Enqueue(request)

	waitRequest := tickler.Request{
		Job: func() error {
			fmt.Println("I am waiting")
			return nil
		},
		Name: "waiting",
	}

	// This will be executed
	scheduler.Enqueue(waitRequest, tickler.IfFailure("hello-world"))
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
	ctx := context.Background()
	limit := tickler.DefaultRequestLimit

	scheduler := tickler.New(ctx, limit)

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
	scheduler.Enqueue(request, tickler.WithRetry(4))
}
```

## Contributing

All kinds of pull request and feature requests are welcomed!

## License

tickler's source code is licensed under [MIT License](https://choosealicense.com/licenses/mit/).