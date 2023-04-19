void F2(void);
void F3(void);
vec4 F5(in vec4 l0);

void F2(void) {
}

void F3(void) {
	F2();
}

vec4 F5(in vec4 l0) {
	F3();
	return vec4(0.0);
}

void main(void) {
	fragColor = F5(gl_FragCoord);
}
