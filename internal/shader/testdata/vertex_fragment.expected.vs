uniform vec2 U0;
attribute vec2 A0;
attribute vec2 A1;
attribute vec4 A2;
varying vec2 V0;
varying vec4 V1;

void main(void) {
	mat4 l0 = mat4(0);
	gl_Position = vec4(0);
	V0 = vec2(0);
	V1 = vec4(0);
	l0 = mat4((2.0) / ((U0).x), 0.0, 0.0, 0.0, 0.0, (2.0) / ((U0).y), 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -1.0, -1.0, 0.0, 1.0);
	gl_Position = (l0) * (vec4(A0, 0.0, 1.0));
	V0 = A1;
	V1 = A2;
	return;
}
