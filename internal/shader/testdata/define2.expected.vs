void F0(out vec2 l0);
void F1(out vec2 l0);

void F0(out vec2 l0) {
	vec2 l1 = vec2(0);
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	vec2 l4 = vec2(0);
	F1(l1);
	l2 = (1.0) * (l1);
	F1(l3);
	l4 = (l3) * (1.0);
	l0 = l2;
	return;
}

void F1(out vec2 l0) {
	l0 = vec2(0.0);
	return;
}
