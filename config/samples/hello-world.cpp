#include <fstream>
int main() {
  std::ofstream out("output/output.txt");
  out << "Hello from SoftwareBuild operator\n";
  return 0;
}
