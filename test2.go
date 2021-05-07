package main

import "sync"

type task struct {
	begin  int
	end    int
	result chan<- int
}

func (t *task) do() {
	sum := 0
	for i := t.begin; i <= t.end; i++ {
		sum += i
	}
	t.result <- sum
}

func InitTask(taskchan chan<- task, r chan int, p int) {
	qu := p / 10
	mod := p % 10
	high := qu * 10
	for i := 0; i < qu; i++ {
		b := 10*i + 1
		e := 10 * (i + 1)
		tsk := task{
			begin:  b,
			end:    e,
			result: r,
		}
		taskchan <- tsk
	}
	if mod != 0 {
		tsk := task{
			begin:  high + 1,
			end:    p,
			result: r,
		}
		taskchan <- tsk
	}
	close(taskchan)
}

func DistributeTask(taskchan <-chan task, wait *sync.WaitGroup, result chan int) {
	wait.Add(1)
}

//func main() {
//	taskchan := make(chan task, 10)
//	resultchan := make(chan int, 10)
//	wait := &sync.WaitGroup{}
//
//	go InitTask(taskchan, resultchan, 100)
//	go DistributeTask(taskchan, wait, resultchan)
//}
