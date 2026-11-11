package engine

import "testing"

// При равном времени SERVICE_END обрабатывается раньше GENERATION.
func TestServiceEndBeforeGenerationAtSameTime(t *testing.T) {
	c := NewEventCalendar()
	c.Schedule(Event{Time: 5, Type: Generation, SourceID: 1})
	c.Schedule(Event{Time: 5, Type: ServiceEnd, DeviceID: 2})
	c.Schedule(Event{Time: 3, Type: Generation, SourceID: 2})
	want := []EventType{Generation, ServiceEnd, Generation}
	for i, w := range want {
		if e := c.PollNext(); e.Type != w {
			t.Fatalf("событие %d: ожидали %v, получили %v", i+1, w, e)
		}
	}
	if !c.IsEmpty() {
		t.Fatal("календарь должен быть пуст")
	}
}
