uniform vec2 U0;
varying vec2 V0;
varying vec4 V1;

void main(void) {
	gl_FragColor = vec4((gl_FragCoord).x, (V0).y, (V1).z, 1.0);
	return;
}
