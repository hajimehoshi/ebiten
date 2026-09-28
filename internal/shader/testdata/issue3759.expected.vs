in vec2 A0;
in vec2 A1;
in vec4 A2;
out vec2 V0;
out vec4 V1;

void main(void) {
	vec2 a0 = A0;
	vec2 a1 = A1;
	vec2 l0 = vec2(0);
	vec2 l1 = vec2(0);
	a0 = (a0) - (vec2(1.0));
	(a1).x = 0.0;
	(a1)[1] = ((a1)[1]) + (1.0);
	l0 = a1;
	l1 = a0;
	a0 = l0;
	a1 = l1;
	gl_Position = vec4(a0, 0.0, 1.0);
	V0 = a1;
	V1 = A2;
	return;
}
