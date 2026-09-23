in vec2 V0;
in vec4 V1;

vec4 F1(in vec4 l0);
vec4 F2(in vec4 l0);
vec4 F3(in vec4 l0, in vec2 l1, in vec4 l2);

vec4 F1(in vec4 l0) {
	return (l0) * (5.0000000000e-01);
}

vec4 F2(in vec4 l0) {
	return (F1(l0)) * (vec4(1.0, 0.0, 0.0, 1.0));
}

vec4 F3(in vec4 l0, in vec2 l1, in vec4 l2) {
	return F2(l2);
}

void main(void) {
	fragColor = F3(gl_FragCoord, V0, V1);
}
