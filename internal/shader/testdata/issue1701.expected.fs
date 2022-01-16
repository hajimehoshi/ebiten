void F2(void);
void F3(void);

void F2(void) {
}

void F3(void) {
	F2();
}

void main(void) {
	F3();
	gl_FragColor = vec4(0.0);
	return;
}
