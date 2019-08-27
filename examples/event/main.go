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
                case event.MouseEnter:  fmt.Printf("MouseEnter: %v: %f %f\n", in, in.X, in.Y)
                case event.MouseLeave:  fmt.Printf("MouseLeave: %v: %f %f\n", in, in.X, in.Y)
                case event.GamepadAxis: fmt.Printf("GamepadAxis: %v: %d %d %f\n", in, in.ID, in.Axis, in.Position)
                case event.GamepadButtonDown: fmt.Printf("GamepadButtonDown: %v: %d %d %f\n", in, in.ID, in.Button, in.Position)
                case event.GamepadButtonUp: fmt.Printf("GamepadButtonUp: %v: %d %d %f\n", in, in.ID, in.Button, in.Position)
                case event.GamepadAttach: fmt.Printf("GamepadAttach: %v: %d %d %d\n", in, in.ID, in.Axes, in.Buttons)
                case event.GamepadDetach: fmt.Printf("GamepadDetach: %v: %d\n", in, in.ID)
                case event.ViewSize: fmt.Printf("ViewSize: %v: (%f %f) (%f %f)\n", in, in.X, in.Y, in.Width, in.Height)
                case event.ViewUpdate: fmt.Printf("ViewUpdate: %v\n", in)
                case event.TouchBegin: fmt.Printf("TouchBegin: %v: %d %f %f\n", in, in.ID, in.X, in.Y)
                case event.TouchMoved: fmt.Printf("TouchMoved: %v: %d %f %f\n", in, in.ID, in.X, in.Y)
                case event.TouchEnd: fmt.Printf("TouchEnd: %v: %d %f %f\n", in, in.ID, in.X, in.Y)
                case event.TouchCancel: fmt.Printf("TouchCancel: %v: %d\n", in, in.ID)
				case CustomEvent: fmt.Printf("Custom Event\n")
				default: fmt.Printf("Not handled: %v\n", queued)
            }
        }
    }
    
    eventQueue := make(chan event.Event)
    
    go receive(eventQueue)
    
    eventQueue <- event.ViewSize{0,0,640,480}
    eventQueue <- event.KeyDown{5, 0}
    eventQueue <- event.KeyCharacter{65, 0, 'A'}
    eventQueue <- event.KeyUp{65, 0}
    eventQueue <- event.MouseEnter{0, 3, 0, 0, 0, 0}
    eventQueue <- event.MouseAxis{7, 8, 0, 7, 5, 0}
    eventQueue <- event.MouseButtonDown{9, 9, 0, 2, 1, 0, 1, 1.0}
    eventQueue <- event.MouseButtonUp{9, 9, 0, 0, 0, 0, 1, 0.0}
    eventQueue <- event.MouseLeave{0, 9, 0, -9, 0, 0}
    eventQueue <- event.GamepadAttach{1, 4, 12}
    eventQueue <- event.GamepadButtonDown{1, 7, 1.0}
    eventQueue <- event.GamepadAxis{1, 2, -1.0}
    eventQueue <- event.GamepadButtonUp{1, 7, 0.0}
    eventQueue <- event.GamepadDetach{1}
    eventQueue <- event.TouchBegin{1, 123, 341, 0, 0, 1.0, true}
    eventQueue <- event.TouchBegin{2, 321, 431, 0, 0, 1.0, false}
    eventQueue <- event.TouchMoved{1, 130, 340, 7, -1, 1.0, true}
    eventQueue <- event.TouchCancel{2}
    eventQueue <- event.TouchEnd{1, 130, 340, 0, 0, 0.0, true}
    eventQueue <- event.ViewUpdate{}
    eventQueue <- CustomEvent{}
    close(eventQueue)
    time.Sleep(time.Second / 10)
    fmt.Printf("done!\n")

}

