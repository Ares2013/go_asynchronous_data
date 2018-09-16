package timer

import (
	"container/heap" // Golang提供的heap库
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

const (
	MIN_TIMER_INTERVAL = 1 * time.Millisecond // 循环定时器的最小时间间隔
)

var (
	nextAddSeq uint = 1 // 用于为每个定时器对象生成一个唯一的递增的序号
)

// 定时器对象
type Timer struct {
	fireTime  time.Time // 触发时间
	interval  time.Duration // 时间间隔（用于循环定时器）
	callback  CallbackFunc // 回调函数
	repeat    bool // 是否循环
	cancelled bool // 是否已经取消
	addseq    uint // 序号
}

// 取消一个定时器，这个定时器将不会被触发
func (t *Timer) Cancel() {
	t.cancelled = true
}

// 判断定时器是否已经取消
func (t *Timer) IsActive() bool {
	return !t.cancelled
}

// 使用一个heap管理所有的定时器
type _TimerHeap struct {
	timers []*Timer
}

// Golang要求heap必须实现下面这些函数，这些函数的含义都是不言自明的

func (h *_TimerHeap) Len() int {
	return len(h.timers)
}

// 使用触发时间和需要对定时器进行比较
func (h *_TimerHeap) Less(i, j int) bool {
	//log.Println(h.timers[i].fireTime, h.timers[j].fireTime)
	t1, t2 := h.timers[i].fireTime, h.timers[j].fireTime
	if t1.Before(t2) {
		return true
	}

	if t1.After(t2) {
		return false
	}
	// t1 == t2, making sure Timer with same deadline is fired according to their add order
	return h.timers[i].addseq < h.timers[j].addseq
}

func (h *_TimerHeap) Swap(i, j int) {
	var tmp *Timer
	tmp = h.timers[i]
	h.timers[i] = h.timers[j]
	h.timers[j] = tmp
}

func (h *_TimerHeap) Push(x interface{}) {
	h.timers = append(h.timers, x.(*Timer))
}

func (h *_TimerHeap) Pop() (ret interface{}) {
	l := len(h.timers)
	h.timers, ret = h.timers[:l-1], h.timers[l-1]
	return
}

// 定时器回调函数的类型定义
type CallbackFunc func()

var (
	timerHeap     _TimerHeap // 定时器heap对象
	timerHeapLock sync.Mutex // 一个全局的锁
)

func init() {
	heap.Init(&timerHeap) // 初始化定时器heap
}

// 设置一个一次性的回调，这个回调将在d时间后触发，并调用callback函数
func AddCallback(d time.Duration, callback CallbackFunc) *Timer {
	t := &Timer{
		fireTime: time.Now().Add(d),
		interval: d,
		callback: callback,
		repeat:   false,
	}
	timerHeapLock.Lock() // 使用锁规避竞争条件
	t.addseq = nextAddSeq
	nextAddSeq += 1

	heap.Push(&timerHeap, t)
	timerHeapLock.Unlock()
	return t
}

// 设置一个定时触发的回调，这个回调将在d时间后第一次触发，以后每隔d时间重复触发，并调用callback函数
func AddTimer(d time.Duration, callback CallbackFunc) *Timer {
	if d < MIN_TIMER_INTERVAL {
		d = MIN_TIMER_INTERVAL
	}

	t := &Timer{
		fireTime: time.Now().Add(d),
		interval: d,
		callback: callback,
		repeat:   true, // 设置为循环定时器
	}
	timerHeapLock.Lock()
	t.addseq = nextAddSeq // set addseq when locked
	nextAddSeq += 1

	heap.Push(&timerHeap, t)
	timerHeapLock.Unlock()
	return t
}

// 对定时器模块进行一次Tick
//
// 一般上层模块需要在一个主线程的goroutine里按一定的时间间隔不停的调用Tick函数，从而确保timer能够按时触发，并且
// 所有Timer的回调函数也在这个goroutine里运行。
func Tick() {
	now := time.Now()
	timerHeapLock.Lock()

	for {
		if timerHeap.Len() <= 0 { // 没有任何定时器，立刻返回
			break
		}

		nextFireTime := timerHeap.timers[0].fireTime
		if nextFireTime.After(now) { // 没有到时间的定时器，返回
			break
		}

		t := heap.Pop(&timerHeap).(*Timer)

		if t.cancelled { // 忽略已经取消的定时器
			continue
		}

		if !t.repeat {
			t.cancelled = true
		}
		// 必须先解锁，然后再调用定时器的回调函数，否则可能导致死锁！！！
			timerHeapLock.Unlock()
		runCallback(t.callback) // 运行回调函数并捕获panic
		timerHeapLock.Lock()

		if t.repeat { // 如果是循环timer就把Timer重新放回heap中
			// add Timer back to heap
			t.fireTime = t.fireTime.Add(t.interval)
			if !t.fireTime.After(now) {
				t.fireTime = now.Add(t.interval)
			}
			t.addseq = nextAddSeq
			nextAddSeq += 1
			heap.Push(&timerHeap, t)
		}
	}
	timerHeapLock.Unlock()
}

// 创建一个goroutine对定时器模块进行定时的Tick
func StartTicks(tickInterval time.Duration) {
	go selfTickRoutine(tickInterval)
}

func selfTickRoutine(tickInterval time.Duration) {
	for {
		time.Sleep(tickInterval)
		Tick()
	}
}

// 运行定时器的回调函数，并捕获panic，将panic转化为错误输出
func runCallback(callback CallbackFunc) {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Callback %v paniced: %v\n", callback, err)
			debug.PrintStack()
		}
	}()
	callback()
}