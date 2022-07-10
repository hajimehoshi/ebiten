int F0(in int l0);
float F1(in float l0);
int F2(void);
float F3(void);

int F0(in int l0) {
	return (1) + (l0);
}

float F1(in float l0) {
	return (1.0) + (l0);
}

int F2(void) {
	return F0(1);
}

float F3(void) {
	return F1(1.0);
}
