void F2(void);
void F3(void);
void F5(in vec4 l0, out vec4 l1);

void F2(void) {
}

void F3(void) {
	F2();
}

void F5(in vec4 l0, out vec4 l1) {
	F3();
	l1 = vec4(0.0);
	return;
}

void main(void) {
	F5(gl_FragCoord, gl_FragData[0]);
}
