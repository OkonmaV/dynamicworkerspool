package dynamicworkerspool

import (
	"errors"
	"time"
)

var ErrTimeout = errors.New("timed out")

type pool struct {
	tasks_ch     chan func()
	tasks_dyn_ch chan func()

	limiter chan struct{}
	ctxdone chan struct{}
	timeout time.Duration
}

func NewPool(min_pool_size, max_pool_size int, worker_annihilation_timeout time.Duration) *pool {
	if min_pool_size > max_pool_size || max_pool_size == 0 {
		panic("pool sizes error: max size is zero or less than min size")
	}
	if max_pool_size != min_pool_size && worker_annihilation_timeout == 0 {
		panic("max pool size is not equal to min size with zero annihilation timeout")
	}
	p := &pool{
		tasks_ch:     make(chan func()),
		tasks_dyn_ch: make(chan func()),
		limiter:      make(chan struct{}, max_pool_size),
		ctxdone:      make(chan struct{}),
		timeout:      worker_annihilation_timeout,
	}
	for i := 0; i < min_pool_size; i++ {
		p.addworker(false)
	}
	return p
}

func (p *pool) addworker(dynamic bool) {
	go func() {
		var timer *time.Timer
		var tsk_ch chan func()
		if dynamic {
			timer = time.NewTimer(p.timeout)
			tsk_ch = p.tasks_dyn_ch
		} else {
			tsk_ch = p.tasks_ch
			timer = &time.Timer{}
		}

	loop:
		for {
			select {
			case <-p.ctxdone:
				break loop
			case <-timer.C:
				break loop
			case task := <-tsk_ch:
				task()
				if dynamic {
					timer.Reset(p.timeout)
				}
			}
		}
		<-p.limiter
	}()
}

// no guaranties of scheduling after pool.Close() called
func (p *pool) Schedule(task func()) {
loop:
	for {
		select {
		case <-p.ctxdone: //TODO: пока выглядит СПОРНО (если закрыли пул и все воркеры вмерли, то без этого будед дед лок, а сейчас нет гарантий выполнения задачи после закрытия пула)
			return
		case p.tasks_ch <- task:
			return
		default:
			select {
			case p.tasks_dyn_ch <- task:
				return
			case p.limiter <- struct{}{}:
				p.addworker(true)
			default:
				continue loop
			}
		}
	}
}

func (p *pool) ScheduleWithTimeout(task func(), timeout time.Duration) error {
	timer := time.NewTimer(timeout)
loop:
	for {
		select {
		case <-p.ctxdone: //TODO: пока выглядит СПОРНО
			return errors.New("pool closed")
		case <-timer.C:
			return ErrTimeout
		case p.tasks_ch <- task:
			timer.Stop()
			return nil
		default:
			select {
			case p.tasks_dyn_ch <- task:
				timer.Stop()
				return nil
			case p.limiter <- struct{}{}:
				p.addworker(true)
			default:
				continue loop
			}
		}
	}
}

func (p *pool) Close() {
	close(p.ctxdone)
}

func (p *pool) Done() {
	for i := 0; i < len(p.limiter); i++ {
		p.limiter <- struct{}{}
	}
}
func (p *pool) DoneWithTimeout(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
loop:
	for i := 0; i < len(p.limiter); i++ {
		select {
		case p.limiter <- struct{}{}:
			continue loop
		case <-timer.C:
			return ErrTimeout
		}
	}
	return nil
}
