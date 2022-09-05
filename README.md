# dynamicworkerspool

## Usage

```go
package main

import (
	"dynamicworkerspool"
	"time"
)

//start do some shit:
func main(){
    pool := dynamicworkerspool.NewPool(2, 5, time.Second*5)
    
    pool.Schedule(func() {
        //do smth
	})
    _ = pool.ScheduleTimeout(func() {
        //do smth
	}, time.Second)

    pool.Close()
    //pool.Done()
    _ = pool.DoneWithTimeout(time.Second)
}
```

