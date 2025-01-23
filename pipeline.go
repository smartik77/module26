package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	bufferSize int = 5                // размер кольцевого буффера
	timeout        = time.Second * 10 // таймаут отправки данных
)

// кольцевой буфер целых чисел
type RingIntBuffer struct {
	array []int
	pos   int
	size  int
	m     sync.Mutex
}

func NewRingIntBuffer(size int) *RingIntBuffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

func (r *RingIntBuffer) Push(el int) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.pos == r.size-1 {
		// Сдвигаем все элементы буфера
		// на одну позицию в сторону начала
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

func (r *RingIntBuffer) Get() []int {
	if r.pos < 0 {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	var output []int = r.array[:r.pos+1]
	// Виртуальная очистка нашего буфера
	r.pos = -1
	return output
}

type StageInt func(<-chan bool, <-chan int) <-chan int

type PipeLineInt struct {
	stages []StageInt
	done   <-chan bool
}

func NewPipelineInt(done <-chan bool, stages ...StageInt) *PipeLineInt {
	return &PipeLineInt{done: done, stages: stages}
}

func (p *PipeLineInt) Run(source <-chan int) <-chan int {
	var c <-chan int = source
	for index := range p.stages {
		c = p.runStageInt(p.stages[index], c)
	}
	return c
}

func (p *PipeLineInt) runStageInt(stage StageInt, sourceChan <-chan int) <-chan int {
	return stage(p.done, sourceChan)
}

func main() {
	dataSource := func() (<-chan int, <-chan bool) {
		source := make(chan int)
		done := make(chan bool)
		go func() {
			defer close(done)
			scanner := bufio.NewScanner(os.Stdin)
			var data string
			for {
				scanner.Scan()
				data = scanner.Text()
				if strings.EqualFold(data, "exit") {
					fmt.Println("Программа завершила работу!")
					return
				}
				i, err := strconv.Atoi(data)
				if err != nil {
					fmt.Println("Программа обрабатывает только целые числа!")
					continue
				}
				source <- i
			}
		}()
		return source, done
	}

	filterNegative := func(done <-chan bool, source <-chan int) <-chan int {
		convertedIntChan := make(chan int)
		go func() {
			for {
				select {
				case data := <-source:
					if data > 0 {
						select {
						case convertedIntChan <- data:
						case <-done:
							return
						}
					}
				case <-done:
					return
				}
			}
		}()
		return convertedIntChan
	}

	filterMultiplesOfThree := func(done <-chan bool, source <-chan int) <-chan int {
		filteredIntChan := make(chan int)
		go func() {
			for {
				select {
				case data := <-source:
					if data != 0 && data%3 == 0 {
						select {
						case filteredIntChan <- data:
						case <-done:
							return
						}
					}
				case <-done:
					return
				}
			}
		}()
		return filteredIntChan
	}

	bufferStageInt := func(done <-chan bool, source <-chan int) <-chan int {
		bufferedIntChan := make(chan int)
		buffer := NewRingIntBuffer(bufferSize)
		go func() {
			for {
				select {
				case data := <-source:
					buffer.Push(data)
				case <-done:
					return
				}
			}
		}()

		go func() {
			for {
				select {
				case <-time.After(timeout):
					bufferData := buffer.Get()
					if bufferData != nil {
						for _, data := range bufferData {
							select {
							case bufferedIntChan <- data:
							case <-done:
								return
							}
						}
					}
				case <-done:
					return
				}
			}
		}()
		return bufferedIntChan
	}

	outputPipeline := func(done <-chan bool, source <-chan int) {
		for {
			select {
			case data := <-source:
				fmt.Printf("Получены данные: %d\n", data)
			case <-done:
				return
			}
		}
	}

	source, done := dataSource()

	pipeline := NewPipelineInt(done, filterNegative, filterMultiplesOfThree, bufferStageInt)
	outputPipeline(done, pipeline.Run(source))
}
