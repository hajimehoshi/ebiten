uniform float U0;
uniform float U1;
uniform float U2;
in vec2 A0;

int F0(in int l0);

int F0(in int l0) {
	return l0;
}

void main(void) {
	int l0 = 0;
	int l2 = 0;
	l0 = 0;
	for (int l1 = 0; l1 < 10; l1++) {
		int l2 = 0;
		l2 = F0(l1);
		l0 = (l0) + (l2);
		for (int l3 = 0; l3 < 10; l3++) {
			int l4 = 0;
			l4 = F0(l3);
			l0 = (l0) + (l4);
		}
	}
	l2 = 0;
	l0 = (l0) + (l2);
	gl_Position = vec4(float(l0));
	return;
}
