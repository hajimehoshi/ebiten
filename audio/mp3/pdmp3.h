#ifndef PDMP3_H
#define PDMP3_H

#include <stddef.h>

//#define DEBUG
#define IMDCT_TABLES
#define IMDCT_NTABLES
#define POW34_ITERATE
#define POW34_TABLE
#define OUTPUT_RAW

#define OK    0
#define ERROR -1

unsigned Get_Byte(void);
unsigned Get_Bytes(unsigned num, unsigned* data_vec);
unsigned Get_Filepos(void);
int Decode_L3(void);
int Read_Frame(void);
size_t writeToWriter(void* data, int size);

#endif
