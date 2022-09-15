package glfwwin

func platformInit() error {
	panic("NOT IMPLEMENTED")
	//@autoreleasepool {
	//
	//    _glfw.ns.helper = [[GLFWHelper alloc] init];
	//
	//    [NSThread detachNewThreadSelector:@selector(doNothing:)
	//                             toTarget:_glfw.ns.helper
	//                           withObject:nil];
	//
	//    [NSApplication sharedApplication];
	//
	//    _glfw.ns.delegate = [[GLFWApplicationDelegate alloc] init];
	//    if (_glfw.ns.delegate == nil)
	//    {
	//        _glfwInputError(GLFW_PLATFORM_ERROR,
	//                        "Cocoa: Failed to create application delegate");
	//        return GLFW_FALSE;
	//    }
	//
	//    [NSApp setDelegate:_glfw.ns.delegate];
	//
	//    NSEvent* (^block)(NSEvent*) = ^ NSEvent* (NSEvent* event)
	//    {
	//        if ([event modifierFlags] & NSEventModifierFlagCommand)
	//            [[NSApp keyWindow] sendEvent:event];
	//
	//        return event;
	//    };
	//
	//    _glfw.ns.keyUpMonitor =
	//        [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyUp
	//                                              handler:block];
	//
	//    if (_glfw.hints.init.ns.chdir)
	//        changeToResourcesDirectory();
	//
	//    // Press and Hold prevents some keys from emitting repeated characters
	//    NSDictionary* defaults = @{@"ApplePressAndHoldEnabled":@NO};
	//    [[NSUserDefaults standardUserDefaults] registerDefaults:defaults];
	//
	//    [[NSNotificationCenter defaultCenter]
	//        addObserver:_glfw.ns.helper
	//           selector:@selector(selectedKeyboardInputSourceChanged:)
	//               name:NSTextInputContextKeyboardSelectionDidChangeNotification
	//             object:nil];
	//
	//    createKeyTables();
	//
	//    _glfw.ns.eventSource = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
	//    if (!_glfw.ns.eventSource)
	//        return GLFW_FALSE;
	//
	//    CGEventSourceSetLocalEventsSuppressionInterval(_glfw.ns.eventSource, 0.0);
	//
	//    if (!initializeTIS())
	//        return GLFW_FALSE;
	//
	//    _glfwPollMonitorsCocoa();
	//
	//    if (![[NSRunningApplication currentApplication] isFinishedLaunching])
	//        [NSApp run];
	//
	//    // In case we are unbundled, make us a proper UI application
	//    if (_glfw.hints.init.ns.menubar)
	//        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	//
	//    return GLFW_TRUE;
	//
	//    } // autoreleasepool
}

func platformTerminate() error {
	panic("NOT IMPLEMENTED")
	//@autoreleasepool {
	//
	//    if (_glfw.ns.inputSource)
	//    {
	//        CFRelease(_glfw.ns.inputSource);
	//        _glfw.ns.inputSource = NULL;
	//        _glfw.ns.unicodeData = nil;
	//    }
	//
	//    if (_glfw.ns.eventSource)
	//    {
	//        CFRelease(_glfw.ns.eventSource);
	//        _glfw.ns.eventSource = NULL;
	//    }
	//
	//    if (_glfw.ns.delegate)
	//    {
	//        [NSApp setDelegate:nil];
	//        [_glfw.ns.delegate release];
	//        _glfw.ns.delegate = nil;
	//    }
	//
	//    if (_glfw.ns.helper)
	//    {
	//        [[NSNotificationCenter defaultCenter]
	//            removeObserver:_glfw.ns.helper
	//                      name:NSTextInputContextKeyboardSelectionDidChangeNotification
	//                    object:nil];
	//        [[NSNotificationCenter defaultCenter]
	//            removeObserver:_glfw.ns.helper];
	//        [_glfw.ns.helper release];
	//        _glfw.ns.helper = nil;
	//    }
	//
	//    if (_glfw.ns.keyUpMonitor)
	//        [NSEvent removeMonitor:_glfw.ns.keyUpMonitor];
	//
	//    _glfw_free(_glfw.ns.clipboardString);
	//
	//    _glfwTerminateNSGL();
	//    _glfwTerminateEGL();
	//    _glfwTerminateOSMesa();
	//
	//    } // autoreleasepool
}
