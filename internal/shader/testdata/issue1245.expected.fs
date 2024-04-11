void F0(in vec4 l0, out vec4 l1);

void F0(in vec4 l0, out vec4 l1) {
	vec4 l2 = vec4(0);
	for (float l3 = 0.0; l3 < 4.0; l3++) {
		(l2).x = ((l2).x) + ((l3) * (1.0000000000e-02));
	}
	l1 = l2;
	return;
}

void main(void) {
	F0(gl_FragCoord, gl_FragData[0]);
}
