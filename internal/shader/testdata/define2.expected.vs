void F0(out vec2 l0) {
	vec2 l1 = vec2(0);
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	vec2 l4 = vec2(0);
	vec2 l5 = vec2(0);
	vec2 l6 = vec2(0);
	F1(l3);
	l2 = (1.0) * (l3);
	F1(l6);
	l5 = (l6) * (1.0);
	l0 = l2;
	return;
}

void F1(out vec2 l0) {
	l0 = vec2(0.0);
	return;
}
