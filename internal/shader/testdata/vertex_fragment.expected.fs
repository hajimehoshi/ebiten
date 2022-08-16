uniform vec2 U0;
varying vec2 V0;
varying vec4 V1;

void F0(in vec4 l0, in vec2 l1, in vec4 l2, out vec4 l3);

void F0(in vec4 l0, in vec2 l1, in vec4 l2, out vec4 l3) {
	l3 = vec4((l0).x, (l1).y, (l2).z, 1.0);
	return;
}

void main(void) {
	vec4 l0 = vec4(0);
	F0(gl_FragCoord, V0, V1, l0);
	gl_FragColor = l0;
}
