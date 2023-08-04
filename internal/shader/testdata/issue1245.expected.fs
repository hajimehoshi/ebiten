vec4 F0(in vec4 l0);

vec4 F0(in vec4 l0) {
	vec4 l1 = vec4(0);
	for (float l2 = 0.0; l2 < 4.0; l2++) {
		(l1).x = ((l1).x) + ((l2) * (1.0000000000e-02));
	}
	return l1;
}

void main(void) {
	fragColor = F0(gl_FragCoord);
}
