package main

import "github.com/hajimehoshi/ebiten/event"
import "fmt"
import "time"

type CustomEvent struct {	
}

func main() {   
    receive := func (queue <- chan event.Event) {
        for { 
            queued := <- queue
            if queued == nil {
				break
			}
            switch in := queued.(type) {
                case event.KeyDown: fmt.Printf("KeyDown: %v: %d %d\n", in, in.Code, in.Modifiers) 
                case event.KeyUp: fmt.Printf("KeyUp: %v: %d %d\n", in, in.Code, in.Modifiers) 
                case event.KeyCharacter: fmt.Printf("KeyChar: %v: %d %d %c\n", in, in.Code, in.Modifiers, in.Character) 
                case event.MouseAxis: fmt.Printf("MouseAxis: %v: %f %f %f\n", in, in.X, in.Y, in.Wheel)
                case event.MouseButtonDown: fmt.Printf("MouseButtonDown: %v: %f %f %d\n", in, in.X, in.Y, in.Button)
                case event.MouseButtonUp: fmt.Printf("MouseButtonUp: %v: %f %f %d\n", in, in.X, in.Y, in.Button)
                case event.GamepadAxis: fmt.Printf("GamepadAxis: %v: %d %d %f\n", in, in.ID, in.Axis, in.Position)
                case event.GamepadButtonDown: fmt.Printf("GamepadButtonDown: %v: %d %d %f\n", in, in.ID, in.Button, in.Position)
                case event.GamepadButtonUp: fmt.Printf("GamepadButtonUp: %v: %d %d %f\n", in, in.ID, in.Button, in.Position)
                case event.GamepadAttach: fmt.Printf("GamepadAttach: %v: %d %d %d\n", in, in.ID, in.Axes, in.Buttons)
                case event.GamepadDetach: fmt.Printf("GamepadDetach: %v: %d\n", in, in.ID)
				case CustomEvent: fmt.Printf("Custom Event\n")
				default: fmt.Printf("Not handled: %v\n", queued)
            }
        }
    }
    
    eventQueue := make(chan event.Event)
    
    go receive(eventQueue)
    
    eventQueue <- event.NewKeyDown(65, 0)
    eventQueue <- event.NewKeyCharacter(65, 0, 'A')
    eventQueue <- event.NewKeyUp(65, 0)
    eventQueue <- event.NewMouseAxis(7, 8, 0, 3, 2, 0)
    eventQueue <- event.NewMouseButtonDown(9, 9, 0, 2, 1, 0, 1, 1.0)
    eventQueue <- event.NewMouseButtonUp(9, 9, 0, 0, 0, 0, 1, 0.0)
    eventQueue <- event.NewGamepadAttach(1, 4, 12)
    eventQueue <- event.NewGamepadButtonDown(1, 7, 1.0)
    eventQueue <- event.NewGamepadAxis(1, 2, -1.0)
    eventQueue <- event.NewGamepadButtonUp(1, 7, 0.0)
    eventQueue <- event.NewGamepadDetach(1)
    eventQueue <- CustomEvent{}
    close(eventQueue)
    time.Sleep(time.Second / 10)
    fmt.Printf("done!\n")

}

