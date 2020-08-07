void main(void) {
	vec4 l0 = vec4(0);
	for (int l1 = 0; l1 < 4; l1++) {
		(l0).x = ((l0).x) + ((l1) * (1.000000000e-02));
	}
	gl_FragColor = l0;
	return;
}
