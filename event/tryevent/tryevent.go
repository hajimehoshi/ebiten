package main

import "github.com/hajimehoshi/ebiten/event"
import "fmt"

func main() {
    toEmit:= []event.Event{}
    toEmit = append(toEmit, event.NewKeyDown(65, 0))
    toEmit = append(toEmit, event.NewKeyCharacter(65, 0, 'A'))
    toEmit = append(toEmit, event.NewKeyUp(65, 0))
    
    
    receive := func (queue <- chan event.Event) {
        for { 
            queued := <- queue
            switch in := queued.(type) {
                case event.KeyDown: fmt.Printf("KeyDown: %v: %d %d\n", in, in.Code, in.Modifiers) 
                case event.KeyUp: fmt.Printf("KeyUp: %v: %d %d\n", in, in.Code, in.Modifiers) 
                case event.KeyCharacter: fmt.Printf("KeyChar: %v: %d %d %c\n", in, in.Code, in.Modifiers, in.Character) 

            default: fmt.Printf("Not handled: %v\n", queued)
            }
        }
    }
    
    eventQueue := make(chan event.Event)
    
    go receive(eventQueue)
    
    for _, evout := range toEmit {
        eventQueue <- evout 
    }
    fmt.Printf("done!\n")

}

