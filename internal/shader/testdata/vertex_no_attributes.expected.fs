in vec4 V0;

vec4 F0(in vec4 l0, in vec4 l1);

vec4 F0(in vec4 l0, in vec4 l1) {
	return l1;
}

void main(void) {
	fragColor = F0(gl_FragCoord, V0);
}
