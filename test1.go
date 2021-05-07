package main

import (
	"math/rand"
)

var lock = make(chan struct{})

func GenerateIntA(done chan struct{}) chan int {
	ch := make(chan int, 5)
	go func() {
	Lable:
		for {
			select {
			case ch <- rand.Int():
			case <-done:
				break Lable
			}
		}
		close(ch)
	}()
	return ch
}

func GenerateIntB(done chan struct{}) chan int {
	ch := make(chan int, 10)
	go func() {
	Lable:
		for {
			select {
			case ch <- rand.Int():
			case <-done:
				break Lable
			}
		}
		close(ch)
	}()
	return ch
}

func GenerateInt(done chan struct{}) chan int {
	ch := make(chan int, 1)
	send := make(chan struct{})
	go func() {
	Lable:
		for {
			select {
			case ch <- <-GenerateIntA(send):
				lock <- struct{}{}
				print("A：")
			case ch <- <-GenerateIntB(send):
				lock <- struct{}{}
				print("B：")
			case <-done:
				send <- struct{}{}
				send <- struct{}{}
				break Lable
			}
		}
		close(ch)
	}()
	return ch
}

//func main() {
//	done := make(chan struct{})
//	ch := GenerateInt(done)
//	//mutex := sync.Mutex{}
//	for i := 0; i < 5; i++ {
//		<-lock
//		fmt.Println(<-ch)
//	}
//}
