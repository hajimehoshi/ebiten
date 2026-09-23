vec4 F0(in vec4 l0);

vec4 F0(in vec4 l0) {
	return vec4(1.0);
}

void main(void) {
	fragColor = F0(gl_FragCoord);
}
