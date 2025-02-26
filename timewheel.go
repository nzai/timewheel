package timewheel

import (
	"sync"
	"time"
)

type TimeWheel struct {
	layers        []*layer
	baseInterval  time.Duration
	slotsPerLayer int
	mu            sync.RWMutex
	keyMap        map[string]*taskEntry
	callback      func(string, any)
	ticker        *time.Ticker
	quit          chan struct{}
}

type layer struct {
	interval   time.Duration
	slots      int
	currentPos int
	buckets    []map[string]*taskEntry
}

type taskEntry struct {
	key        string
	value      any
	expiration time.Time
	layerIndex int
	bucketPos  int
	rounds     int
}

func NewTimeWheel(baseInterval time.Duration, slotsPerLayer int, callback func(key string, value any)) *TimeWheel {
	tw := &TimeWheel{
		baseInterval:  baseInterval,
		slotsPerLayer: slotsPerLayer,
		keyMap:        make(map[string]*taskEntry),
		callback:      callback,
		ticker:        time.NewTicker(baseInterval),
		quit:          make(chan struct{}),
	}

	// Initialize layers
	tw.addLayer(baseInterval)
	tw.addLayer(baseInterval * time.Duration(slotsPerLayer))
	tw.addLayer(baseInterval * time.Duration(slotsPerLayer*slotsPerLayer))

	go tw.run()
	return tw
}

func (tw *TimeWheel) addLayer(interval time.Duration) {
	l := &layer{
		interval:   interval,
		slots:      tw.slotsPerLayer,
		currentPos: 0,
		buckets:    make([]map[string]*taskEntry, tw.slotsPerLayer),
	}
	for i := 0; i < tw.slotsPerLayer; i++ {
		l.buckets[i] = make(map[string]*taskEntry)
	}
	tw.layers = append(tw.layers, l)
}

func (tw *TimeWheel) run() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tick()
		case <-tw.quit:
			tw.ticker.Stop()
			return
		}
	}
}

func (tw *TimeWheel) tick() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	now := time.Now()
	prevPositions := make([]int, len(tw.layers))
	for i, l := range tw.layers {
		prevPositions[i] = l.currentPos
	}

	// Update position for base layer
	baseLayer := tw.layers[0]
	baseLayer.currentPos = (baseLayer.currentPos + 1) % baseLayer.slots
	tw.processLayer(baseLayer, now)

	// Check and update higher layers
	for i := 1; i < len(tw.layers); i++ {
		prevLayer := tw.layers[i-1]
		currentLayer := tw.layers[i]
		if prevPositions[i-1] == prevLayer.slots-1 {
			currentLayer.currentPos = (currentLayer.currentPos + 1) % currentLayer.slots
			tw.processLayer(currentLayer, now)
		}
	}
}

func (tw *TimeWheel) processLayer(l *layer, now time.Time) {
	bucket := l.buckets[l.currentPos]
	for key, entry := range bucket {
		if entry.rounds > 0 {
			entry.rounds--
			continue
		}

		if entry.expiration.After(now) {
			d := entry.expiration.Sub(now)
			targetLayer, targetPos, rounds := tw.findPosition(d)
			if targetLayer == nil {
				if tw.callback != nil {
					go tw.callback(entry.key, entry.value)
				}
				delete(tw.keyMap, key)
				delete(bucket, key)
				continue
			}

			delete(bucket, key)
			entry.layerIndex = tw.getLayerIndex(targetLayer)
			entry.bucketPos = targetPos
			entry.rounds = rounds
			targetLayer.buckets[targetPos][key] = entry
		} else {
			if tw.callback != nil {
				go tw.callback(entry.key, entry.value)
			}
			delete(tw.keyMap, key)
			delete(bucket, key)
		}
	}
}

func (tw *TimeWheel) findPosition(d time.Duration) (*layer, int, int) {
	for i := len(tw.layers) - 1; i >= 0; i-- {
		l := tw.layers[i]
		if l.interval <= d {
			slots := int(d / l.interval)
			rounds := slots / l.slots
			pos := (l.currentPos + slots) % l.slots
			return l, pos, rounds
		}
	}
	return nil, 0, 0
}

func (tw *TimeWheel) getLayerIndex(target *layer) int {
	for i, l := range tw.layers {
		if l == target {
			return i
		}
	}
	return -1
}

func (tw *TimeWheel) Set(key string, value any, expiration time.Duration) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	now := time.Now()
	expireAt := now.Add(expiration)

	if entry, exists := tw.keyMap[key]; exists {
		delete(tw.keyMap, key)
		tw.layers[entry.layerIndex].buckets[entry.bucketPos][key] = nil
		delete(tw.layers[entry.layerIndex].buckets[entry.bucketPos], key)
	}

	if expiration <= 0 {
		if tw.callback != nil {
			go tw.callback(key, value)
		}
		return
	}

	d := expiration
	targetLayer, targetPos, rounds := tw.findPosition(d)
	if targetLayer == nil {
		if tw.callback != nil {
			go tw.callback(key, value)
		}
		return
	}

	entry := &taskEntry{
		key:        key,
		value:      value,
		expiration: expireAt,
		layerIndex: tw.getLayerIndex(targetLayer),
		bucketPos:  targetPos,
		rounds:     rounds,
	}

	targetLayer.buckets[targetPos][key] = entry
	tw.keyMap[key] = entry
}

func (tw *TimeWheel) Delete(key string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	entry, exists := tw.keyMap[key]
	if !exists {
		return
	}

	delete(tw.keyMap, key)
	layer := tw.layers[entry.layerIndex]
	delete(layer.buckets[entry.bucketPos], key)
}

func (tw *TimeWheel) Move(key string, expiration time.Duration) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	entry, exists := tw.keyMap[key]
	if !exists {
		return
	}

	now := time.Now()
	newExpireAt := now.Add(expiration)
	d := expiration

	oldLayer := tw.layers[entry.layerIndex]
	delete(oldLayer.buckets[entry.bucketPos], key)

	if d <= 0 {
		if tw.callback != nil {
			go tw.callback(entry.key, entry.value)
		}
		delete(tw.keyMap, key)
		return
	}

	targetLayer, targetPos, rounds := tw.findPosition(d)
	if targetLayer == nil {
		if tw.callback != nil {
			go tw.callback(entry.key, entry.value)
		}
		delete(tw.keyMap, key)
		return
	}

	entry.expiration = newExpireAt
	entry.layerIndex = tw.getLayerIndex(targetLayer)
	entry.bucketPos = targetPos
	entry.rounds = rounds
	targetLayer.buckets[targetPos][key] = entry
}

func (tw *TimeWheel) FlushAll() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.keyMap = make(map[string]*taskEntry)
	for _, l := range tw.layers {
		for i := range l.buckets {
			l.buckets[i] = make(map[string]*taskEntry)
		}
	}
}

func (tw *TimeWheel) Stop() {
	close(tw.quit)
}
