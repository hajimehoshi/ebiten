uniform vec2 U0;
in vec2 V0;
in vec4 V1;

vec4 F0(in vec4 l0, in vec2 l1, in vec4 l2);

vec4 F0(in vec4 l0, in vec2 l1, in vec4 l2) {
	return vec4((l0).x, (l1).y, (l2).z, 1.0);
}

void main(void) {
	fragColor = F0(gl_FragCoord, V0, V1);
}
