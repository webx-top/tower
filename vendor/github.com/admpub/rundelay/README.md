# run delay

延迟运行。
1. 在延迟时间内以最后的一个 Run 的运行时间顺延延迟时间，并保证在并发执行Run时，延迟执行的函数只执行一次
2. 超过延迟时间之后，只要没有调用过 Close 函数，可以继续调用 Run 函数启动新的延迟
3. 可以调用 Done 函数获取延迟函数的执行结果

## 例子
```go
import (
    "time"

    "github.com/admpub/rundelay"
)

func main(){
	delay := time.Second * 2
	dr := rundelay.New(delay, func(v string) error{
        return nil
    })
    defer dr.Close()

    // ...

    dr.Run(`test`) // 可以多次调用使用此代码且并发执行
    // 返回结果为bool值, true代表启动新的延迟 hasRun := dr.Run(`test`)

    // ...

    err := dr.Done() // 获取执行结果
    if err != nil {
        panic(err)
    }
    
}
```
[实例](rundelay_test.go)