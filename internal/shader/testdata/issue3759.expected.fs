in vec2 V0;
in vec4 V1;

vec4 F0(in vec4 l0, in vec2 l1, in vec4 l2);

vec4 F0(in vec4 l0, in vec2 l1, in vec4 l2) {
	l0 = vec4(0.0);
	(l1).x = ((l1).x) + (1.0);
	(l2)[3] = ((l2)[3]) * (2.0);
	l2 = (l2) - (1.0);
	return vec4((l0).x, (l1).y, (l2).z, (l2).w);
}

void main(void) {
	fragColor = F0(gl_FragCoord, V0, V1);
}
