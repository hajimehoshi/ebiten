uniform float U0;
uniform float U1;
uniform float U2;
attribute vec2 A0;

void F0(in int l0, out int l1);

void F0(in int l0, out int l1) {
	l1 = l0;
	return;
}

void main(void) {
	int l0 = 0;
	int l2 = 0;
	l0 = 0;
	for (int l1 = 0; l1 < 10; l1++) {
		int l2 = 0;
		int l3 = 0;
		F0(l1, l2);
		l3 = l2;
		l0 = (l0) + (l3);
		for (int l4 = 0; l4 < 10; l4++) {
			int l5 = 0;
			int l6 = 0;
			F0(l4, l5);
			l6 = l5;
			l0 = (l0) + (l6);
		}
	}
	l2 = 0;
	l0 = (l0) + (l2);
	gl_Position = vec4(l0);
	return;
}
