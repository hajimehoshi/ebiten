attribute vec2 A0;

void F0(void);
void F1(void);
void F2(void);
void F3(void);

void F0(void) {
	F1();
	F2();
}

void F1(void) {
	F2();
	F3();
}

void F2(void) {
}

void F3(void) {
	F2();
}

void main(void) {
	F0();
	gl_Position = vec4(0.0);
	return;
}
