void F0(in vec2 l0, out vec2 l1);
void F1(in vec2 l0, out vec2 l1);

void F0(in vec2 l0, out vec2 l1) {
	vec2 l2 = vec2(0);
	F1(l0, l2);
	l1 = l2;
	return;
}

void F1(in vec2 l0, out vec2 l1) {
	l1 = l0;
	return;
}
