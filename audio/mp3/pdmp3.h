#ifndef PDMP3_H
#define PDMP3_H

#include <stddef.h>
#include <stdint.h>

//#define DEBUG
#define IMDCT_TABLES
#define IMDCT_NTABLES
#define POW34_ITERATE
#define POW34_TABLE
#define OUTPUT_RAW

#define OK    0
#define ERROR -1
#define TRUE       1
#define FALSE      0

unsigned Get_Byte(void);
unsigned Get_Bytes(unsigned num, unsigned* data_vec);
unsigned Get_Filepos(void);
int Decode_L3(void);
int Read_Frame(void);
size_t writeToWriter(void* data, int size);

int Get_Main_Data(unsigned main_data_size,unsigned main_data_begin);
unsigned Get_Main_Bit(void);
unsigned Get_Main_Bits(unsigned number_of_bits);
unsigned Get_Main_Pos(void);
int Set_Main_Pos(unsigned bit_pos);
unsigned Get_Main_Bit(void);
unsigned Get_Main_Bits(unsigned number_of_bits);

void Get_Sideinfo(unsigned sideinfo_size);
unsigned Get_Side_Bits(unsigned number_of_bits);

int Read_CRC(void);

int Huffman_Decode(unsigned table_num, int32_t* x, int32_t*y, int32_t* v, int32_t* w);

#endif
