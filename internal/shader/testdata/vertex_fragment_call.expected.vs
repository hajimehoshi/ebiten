in vec2 A0;
in vec2 A1;
in vec4 A2;
out vec2 V0;
out vec4 V1;

vec4 F0(in vec2 l0);
vec4 F1(in vec4 l0);

vec4 F0(in vec2 l0) {
	return vec4(l0, 0.0, 1.0);
}

vec4 F1(in vec4 l0) {
	return (l0) * (5.0000000000e-01);
}

void main(void) {
	gl_Position = vec4(0);
	V0 = vec2(0);
	V1 = vec4(0);
	gl_Position = F0(A0);
	V0 = A1;
	V1 = F1(A2);
	return;
}
