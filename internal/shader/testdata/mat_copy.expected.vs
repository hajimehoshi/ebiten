mat2 F0(in mat2 l0);
mat3 F1(in mat3 l0);
mat4 F2(in mat4 l0);
void F3(in float l0, out mat2 l1, out mat3 l2, out mat4 l3);

mat2 F0(in mat2 l0) {
	return l0;
}

mat3 F1(in mat3 l0) {
	return l0;
}

mat4 F2(in mat4 l0) {
	return l0;
}

void F3(in float l0, out mat2 l1, out mat3 l2, out mat4 l3) {
	l1 = mat2(l0);
	l2 = mat3(l0);
	l3 = mat4(l0);
	return;
}
