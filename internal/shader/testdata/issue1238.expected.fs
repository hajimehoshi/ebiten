void F0(in vec4 l0, out vec4 l1);

void F0(in vec4 l0, out vec4 l1) {
	if (true) {
		l1 = l0;
		return;
	}
	l1 = l0;
	return;
}

void main(void) {
	F0(gl_FragCoord, gl_FragData[0]);
}
