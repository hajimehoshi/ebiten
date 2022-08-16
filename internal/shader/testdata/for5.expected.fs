uniform float U0;
uniform float U1;
uniform float U2;

int F0(in int l0);
void F1(in vec4 l0, out vec4 l1);

int F0(in int l0) {
	return l0;
}

void F1(in vec4 l0, out vec4 l1) {
	int l2 = 0;
	int l4 = 0;
	l2 = 0;
	for (int l3 = 0; l3 < 10; l3++) {
		int l4 = 0;
		l4 = F0(l3);
		l2 = (l2) + (l4);
		for (int l5 = 0; l5 < 10; l5++) {
			int l6 = 0;
			l6 = F0(l5);
			l2 = (l2) + (l6);
		}
	}
	l4 = 0;
	l2 = (l2) + (l4);
	l1 = vec4(l2);
	return;
}

void main(void) {
	vec4 l0 = vec4(0);
	F1(gl_FragCoord, l0);
	gl_FragColor = l0;
}
