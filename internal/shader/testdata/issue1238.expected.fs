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
	vec4 l0 = vec4(0);
	F0(gl_FragCoord, l0);
	gl_FragColor = l0;
}
